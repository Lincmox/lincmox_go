package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist
var webDistFS embed.FS

// WebUI returns a file system rooted at dist for serving the Web UI.
func WebUI() http.FileSystem {
	sub, err := fs.Sub(webDistFS, "dist")
	if err != nil {
		panic(err)
	}
	return http.FS(sub)
}