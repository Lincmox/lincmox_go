package main

import (
	"net/http"

	"github.com/lincmox/lincmox/internal/web"
)

// webUI returns a file system rooted at web/dist for serving the Web UI.
func webUI() http.FileSystem {
	return web.WebUI()
}