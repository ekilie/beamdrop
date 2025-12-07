package handlers

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/tachRoutine/beamdrop-go/pkg/logger"
	"github.com/tachRoutine/beamdrop-go/static"
)

func StaticHandler(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	if urlPath == "/" {
		urlPath = "/index.html"
	}

	logger.Debug("Serving static file: %s", urlPath)
	file, err := static.FrontendFiles.Open("frontend/dist" + urlPath)
	if err != nil {
		logger.Warn("Static file not found: %s", urlPath)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Not found"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(path.Ext(urlPath))
	if mimeType := mime.TypeByExtension(ext); mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	io.Copy(w, file)
}

