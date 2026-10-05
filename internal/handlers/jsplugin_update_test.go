package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"songloft/internal/database/testutil"
	"songloft/internal/jsplugin"
)

func updateTestZIP(t *testing.T, manifest *jsplugin.PluginManifest) []byte {
	t.Helper()
	code := []byte("function onInit() {}\nfunction onDeinit() {}")
	hash := sha256.Sum256(code)
	manifest.EntryHash = hex.EncodeToString(hash[:])
	build := func() []byte {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		data, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range []struct {
			name string
			data []byte
		}{{"plugin.json", data}, {"main.js", code}} {
			w, err := zw.Create(file.name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write(file.data); err != nil {
				t.Fatal(err)
			}
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}
	zipHash, err := jsplugin.ComputeCanonicalZipHash(build())
	if err != nil {
		t.Fatal(err)
	}
	manifest.ZipHash = zipHash
	return build()
}

// 模拟服务端正在下载时客户端离开页面：安装仍完成，且运行中的插件必须同步更新。
func TestPluginUpdateReloadAfterClientDisconnect(t *testing.T) {
	for _, batch := range []bool{false, true} {
		t.Run(fmt.Sprintf("batch=%t", batch), func(t *testing.T) {
			db := testutil.OpenMemoryDB(t)
			repo := db.JSPluginRepository()
			pluginsDir, dataDir := filepath.Join(t.TempDir(), "plugins"), t.TempDir()
			manager := jsplugin.NewManager(repo, pluginsDir, dataDir, "", nil, db)
			t.Cleanup(func() { _ = manager.Close() })
			packager := manager.Packager()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var updatedZIP []byte
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/manifest" {
					_ = json.NewEncoder(w).Encode(jsplugin.PluginManifest{Version: "2.0.0", DownloadURL: "http://" + r.Host + "/plugin.zip"})
					return
				}
				cancel()
				_, _ = w.Write(updatedZIP)
			}))
			defer upstream.Close()
			manifest := &jsplugin.PluginManifest{Name: "Disconnect test", Version: "1.0.0", Author: "test", EntryPath: "disconnect-test", Main: "main.js", UpdateURL: upstream.URL + "/manifest", Permissions: []string{}}
			plugin, _, err := packager.InstallFromUpload(updateTestZIP(t, manifest))
			if err != nil {
				t.Fatal(err)
			}
			if err := manager.LoadPlugin(context.Background(), plugin); err != nil {
				t.Fatal(err)
			}
			manifest.Version = "2.0.0"
			updatedZIP = updateTestZIP(t, manifest)
			handler := NewJSPluginHandler(packager, repo, manager, nil, nil, db)
			path := fmt.Sprintf("/api/v1/jsplugins/%d/update", plugin.ID)
			if batch {
				path = "/api/v1/jsplugins/update-all"
			}
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}")).WithContext(ctx)
			router := chi.NewRouter()
			handler.RegisterRoutes(router)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if ctx.Err() != context.Canceled {
				t.Fatal("request was not canceled during download")
			}
			if response.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			stored, err := repo.GetByID(context.Background(), plugin.ID)
			if err != nil {
				t.Fatal(err)
			}
			if stored.Version != "2.0.0" {
				t.Fatalf("stored version=%s", stored.Version)
			}
			running, ok := manager.GetService(plugin.EntryPath)
			if !ok || running.Plugin().Version != "2.0.0" {
				t.Fatalf("new version was installed but not loaded: service=%v", running)
			}
		})
	}
}

// 上传包已经被读取后客户端断开，新安装的插件仍须启用，覆盖安装仍须重载。
func TestUploadedPluginActivationAfterClientDisconnect(t *testing.T) {
	for _, overwrite := range []bool{false, true} {
		t.Run(fmt.Sprintf("overwrite=%t", overwrite), func(t *testing.T) {
			db := testutil.OpenMemoryDB(t)
			repo := db.JSPluginRepository()
			manager := jsplugin.NewManager(repo, filepath.Join(t.TempDir(), "plugins"), t.TempDir(), "", nil, db)
			t.Cleanup(func() { _ = manager.Close() })
			manifest := &jsplugin.PluginManifest{Name: "Upload test", Version: "1.0.0", Author: "test", EntryPath: "upload-test", Main: "main.js", Permissions: []string{}}
			if overwrite {
				plugin, _, err := manager.Packager().InstallFromUpload(updateTestZIP(t, manifest))
				if err != nil {
					t.Fatal(err)
				}
				if err := manager.LoadPlugin(context.Background(), plugin); err != nil {
					t.Fatal(err)
				}
			}
			manifest.Version = "2.0.0"
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("file", "plugin.zip")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := part.Write(updateTestZIP(t, manifest)); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/jsplugins/upload", &body).WithContext(ctx)
			request.Header.Set("Content-Type", writer.FormDataContentType())
			response := httptest.NewRecorder()
			handler := NewJSPluginHandler(manager.Packager(), repo, manager, nil, nil, db)
			handler.handleUpload(response, request)
			wantStatus := http.StatusCreated
			if overwrite {
				wantStatus = http.StatusOK
			}
			if response.Code != wantStatus {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			running, ok := manager.GetService(manifest.EntryPath)
			if !ok || running.Plugin().Version != "2.0.0" {
				t.Fatal("installed plugin was not activated")
			}
		})
	}
}
