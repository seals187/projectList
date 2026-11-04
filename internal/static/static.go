// Package static serves the embedded SvelteKit frontend.
//
// Precompressed variants (.br/.gz, generated at build time) are served
// directly – no compression at runtime, no CPU overhead.
// Hashed files under _app/immutable/ are cached for one year, everything
// else is revalidated via ETag. Unknown paths without a file extension
// get index.html (SPA fallback).
package static

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"
)

type file struct {
	etag string
	br   bool
	gz   bool
}

type Handler struct {
	fsys  fs.FS
	files map[string]*file
}

var extraTypes = map[string]string{
	".ico":         "image/x-icon",
	".woff2":       "font/woff2",
	".txt":         "text/plain; charset=utf-8",
	".webmanifest": "application/manifest+json",
	".js":          "text/javascript; charset=utf-8",
	".css":         "text/css; charset=utf-8",
	".html":        "text/html; charset=utf-8",
	".svg":         "image/svg+xml",
	".json":        "application/json",
}

// New indexes all files in fsys and computes ETags.
func New(fsys fs.FS) (*Handler, error) {
	h := &Handler{fsys: fsys, files: map[string]*file{}}
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasSuffix(p, ".gitkeep") {
			return err
		}
		if strings.HasSuffix(p, ".br") || strings.HasSuffix(p, ".gz") {
			return nil
		}
		f, err := fsys.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		sum := sha256.New()
		if _, err := io.Copy(sum, f); err != nil {
			return err
		}
		h.files[p] = &file{etag: `"` + hex.EncodeToString(sum.Sum(nil))[:16] + `"`}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for p, f := range h.files {
		_, errBr := fs.Stat(fsys, p+".br")
		_, errGz := fs.Stat(fsys, p+".gz")
		f.br, f.gz = errBr == nil, errGz == nil
	}
	return h, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	p := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if p == "" {
		p = "index.html"
	}
	if _, ok := h.files[p]; !ok {
		ext := path.Ext(p)
		if ext != "" && ext != ".html" {
			http.NotFound(w, r)
			return
		}
		p = "index.html" // SPA fallback
	}
	f, ok := h.files[p]
	if !ok {
		http.Error(w, "Frontend nicht gebaut (web/build fehlt).", http.StatusNotFound)
		return
	}

	hdr := w.Header()
	ct := extraTypes[path.Ext(p)]
	if ct == "" {
		ct = mime.TypeByExtension(path.Ext(p))
	}
	if ct == "" {
		ct = "application/octet-stream"
	}
	hdr.Set("Content-Type", ct)
	hdr.Set("Vary", "Accept-Encoding")
	if strings.HasPrefix(p, "_app/immutable/") {
		hdr.Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		hdr.Set("Cache-Control", "no-cache")
	}

	name, etag := p, f.etag
	ae := r.Header.Get("Accept-Encoding")
	switch {
	case f.br && strings.Contains(ae, "br"):
		name, etag = p+".br", strings.TrimSuffix(etag, `"`)+`-br"`
		hdr.Set("Content-Encoding", "br")
	case f.gz && strings.Contains(ae, "gzip"):
		name, etag = p+".gz", strings.TrimSuffix(etag, `"`)+`-gz"`
		hdr.Set("Content-Encoding", "gzip")
	}
	hdr.Set("ETag", etag)

	fh, err := h.fsys.Open(name)
	if err != nil {
		http.Error(w, "Interner Fehler", http.StatusInternalServerError)
		return
	}
	defer fh.Close()
	rs, ok := fh.(io.ReadSeeker)
	if !ok {
		http.Error(w, "Interner Fehler", http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, "", time.Time{}, rs)
}
