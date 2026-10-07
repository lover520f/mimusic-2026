package app

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"testing/fstest"

	"github.com/andybalholm/brotli"
)

func TestPrecompressedFSCustomEntry(t *testing.T) {
	original := []byte(`<base href="/"><script src="equalizer.js"></script>`)
	modified := []byte(`<base href="/xxx/"><script src="equalizer.js"></script>`)
	distFS := fstest.MapFS{"index.html": &fstest.MapFile{Data: original}}
	pfs := newPrecompressedFS(distFS)
	pfs.addCustomEntry("index.html", modified, ".html")

	// 旧版 identity 响应返回 original，却使用 modified 的校验值；升级必须刷新它。
	legacyRequest := httptest.NewRequest(http.MethodGet, "/xxx/", nil)
	legacyRequest.Header.Set("If-None-Match", fmt.Sprintf(`"%08x"`, crc32.ChecksumIEEE(modified)))
	legacyResponse := httptest.NewRecorder()
	if !pfs.serve(legacyResponse, legacyRequest, "index.html") || legacyResponse.Code != http.StatusOK {
		t.Errorf("legacy cache must be refreshed: status = %d", legacyResponse.Code)
	}
	if got := readStaticResponse(t, legacyResponse); !bytes.Equal(got, modified) {
		t.Errorf("legacy cache refresh body = %q, want %q", got, modified)
	}

	for _, encoding := range []string{"", "identity", "gzip", "br"} {
		t.Run("encoding="+encoding, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/xxx/", nil)
			req.Header.Set("Accept-Encoding", encoding)
			rec := httptest.NewRecorder()
			if !pfs.serve(rec, req, "index.html") {
				t.Fatal("custom entry was not served")
			}
			wantEncoding := encoding
			if wantEncoding == "identity" {
				wantEncoding = ""
			}
			if got := rec.Header().Get("Content-Encoding"); got != wantEncoding {
				t.Fatalf("Content-Encoding = %q, want %q", got, wantEncoding)
			}
			if got := rec.Header().Get("Content-Length"); got != strconv.Itoa(rec.Body.Len()) {
				t.Errorf("Content-Length = %q, body size = %d", got, rec.Body.Len())
			}
			if got := readStaticResponse(t, rec); !bytes.Equal(got, modified) {
				t.Errorf("body = %q, want modified HTML %q", got, modified)
			}

			// 条件请求必须仍能校验这份修改后的页面。
			req.Header.Set("If-None-Match", rec.Header().Get("ETag"))
			cached := httptest.NewRecorder()
			if !pfs.serve(cached, req, "index.html") || cached.Code != http.StatusNotModified || cached.Body.Len() != 0 {
				t.Errorf("conditional request: status = %d, body size = %d", cached.Code, cached.Body.Len())
			}
		})
	}
}

func TestPrecompressedFSOriginalEntry(t *testing.T) {
	raw := []byte("console.log('original static file');")
	var br, gz bytes.Buffer
	brw := brotli.NewWriter(&br)
	if _, err := brw.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := brw.Close(); err != nil {
		t.Fatal(err)
	}
	gzw := gzip.NewWriter(&gz)
	if _, err := gzw.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatal(err)
	}
	distFS := fstest.MapFS{
		"equalizer.js":    &fstest.MapFile{Data: raw},
		"equalizer.js.br": &fstest.MapFile{Data: br.Bytes()},
		"equalizer.js.gz": &fstest.MapFile{Data: gz.Bytes()},
	}
	pfs := newPrecompressedFS(distFS)
	for _, encoding := range []string{"", "gzip", "br"} {
		t.Run("encoding="+encoding, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/xxx/equalizer.js", nil)
			req.Header.Set("Accept-Encoding", encoding)
			rec := httptest.NewRecorder()
			if !pfs.serve(rec, req, "equalizer.js") {
				t.Fatal("original entry was not served")
			}
			if got := readStaticResponse(t, rec); !bytes.Equal(got, raw) {
				t.Errorf("body = %q, want original file %q", got, raw)
			}
		})
	}
}

func readStaticResponse(t *testing.T, rec *httptest.ResponseRecorder) []byte {
	t.Helper()
	var reader io.Reader = rec.Body
	switch rec.Header().Get("Content-Encoding") {
	case "br":
		reader = brotli.NewReader(rec.Body)
	case "gzip":
		gzr, err := gzip.NewReader(rec.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer gzr.Close()
		reader = gzr
	}
	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
