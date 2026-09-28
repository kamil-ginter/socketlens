package webui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var embedded embed.FS

func Handler() http.Handler {
	staticFS, err := fs.Sub(embedded, "static")
	if err != nil {
		panic(err)
	}

	return http.FileServer(http.FS(staticFS))
}
