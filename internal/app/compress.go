package app

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"hash/crc32"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/andybalholm/brotli"
)

type compressedEntry struct {
	raw      []byte // 运行时修改后的原始内容；构建时预压缩文件仍从 distFS 读取
	br       []byte
	gz       []byte
	mimeType string
	etag     string
}

type precompressedFS struct {
	entries map[string]*compressedEntry
	distFS  fs.FS
}

// newPrecompressedFS 从 embed.FS 中加载构建时生成的 .br/.gz 预压缩文件。
// 若构建时未预压缩（brotli CLI 未安装），entries 为空，所有请求 fallback 到 http.FileServer + chi gzip。
func newPrecompressedFS(distFS fs.FS) *precompressedFS {
	pfs := &precompressedFS{entries: make(map[string]*compressedEntry), distFS: distFS}

	var brTotal, gzTotal int64

	fs.WalkDir(distFS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".br") {
			return err
		}

		basePath := strings.TrimSuffix(p, ".br")
		brBytes, err := fs.ReadFile(distFS, p)
		if err != nil {
			return nil
		}

		ext := strings.ToLower(path.Ext(basePath))
		mimeType := mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		entry := &compressedEntry{
			br:       brBytes,
			mimeType: mimeType,
			etag:     fmt.Sprintf(`"%08x"`, crc32.ChecksumIEEE(brBytes)),
		}

		if gzBytes, err := fs.ReadFile(distFS, basePath+".gz"); err == nil {
			entry.gz = gzBytes
			gzTotal += int64(len(gzBytes))
		}

		pfs.entries[basePath] = entry
		brTotal += int64(len(brBytes))
		return nil
	})

	if len(pfs.entries) > 0 {
		slog.Info("加载预压缩静态资源",
			"文件数", len(pfs.entries),
			"brotli", formatBytes(brTotal),
			"gzip", formatBytes(gzTotal),
		)
	}
	return pfs
}

// addCustomEntry 对运行时修改过的文件（如 base-path 注入后的 index.html）重新压缩。
func (pfs *precompressedFS) addCustomEntry(p string, raw []byte, ext string) {
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	var brBuf bytes.Buffer
	brw := brotli.NewWriterLevel(&brBuf, 6)
	brw.Write(raw)
	brw.Close()

	var gzBuf bytes.Buffer
	gzw, _ := gzip.NewWriterLevel(&gzBuf, gzip.BestCompression)
	gzw.Write(raw)
	gzw.Close()

	pfs.entries[p] = &compressedEntry{
		raw:      bytes.Clone(raw),
		br:       brBuf.Bytes(),
		gz:       gzBuf.Bytes(),
		mimeType: mimeType,
		// 旧版 identity 返回原始页面却使用修改后的校验值；区分旧 ETag，避免升级后继续命中错误缓存。
		etag: fmt.Sprintf(`"custom-%08x"`, crc32.ChecksumIEEE(raw)),
	}
}

// serve 尝试从预压缩缓存中服务请求，返回 true 表示已处理。
func (pfs *precompressedFS) serve(w http.ResponseWriter, r *http.Request, filePath string) bool {
	entry, ok := pfs.entries[filePath]
	if !ok {
		return false
	}

	w.Header().Set("ETag", entry.etag)
	w.Header().Set("Vary", "Accept-Encoding")

	if match := r.Header.Get("If-None-Match"); match != "" {
		if strings.Contains(match, entry.etag) {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
	}

	w.Header().Set("Content-Type", entry.mimeType)

	accept := r.Header.Get("Accept-Encoding")
	if strings.Contains(accept, "br") && len(entry.br) > 0 {
		w.Header().Set("Content-Encoding", "br")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(entry.br)))
		w.Write(entry.br)
		return true
	}
	if strings.Contains(accept, "gzip") && len(entry.gz) > 0 {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(entry.gz)))
		w.Write(entry.gz)
		return true
	}

	// 运行时修改的文件必须使用修改后的原文，不能回退到 embed.FS 中的旧内容。
	raw := entry.raw
	if raw == nil {
		var err error
		raw, err = fs.ReadFile(pfs.distFS, filePath)
		if err != nil {
			return false
		}
	}
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(raw)))
	w.Write(raw)
	return true
}

func formatBytes(b int64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.1fMB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1fKB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%dB", b)
	}
}
