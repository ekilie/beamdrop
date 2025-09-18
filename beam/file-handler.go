package beam

// NOTE: This file contains a duplicate implementation of the file listing functionality
// The main implementation is now in server.go as part of the /files endpoint handler
// This file can be safely removed as it's no longer used

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"time"
)

func FilesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	reqPath := r.URL.Query().Get("path")
	// FIXME:Note: This function needs sharedDir parameter to work properly
	// It should be refactored to accept sharedDir as a parameter
	target, err := ResolvePath(".", reqPath) // Using current directory as fallback
	if err != nil {
		http.Error(w, `{"error":"invalid path"}`, http.StatusBadRequest)
		return
	}

	files, err := os.ReadDir(target)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
		return
	}

	var fileList []File
	for _, f := range files {
		info, err := f.Info()
		if err != nil {
			continue
		}
		fileList = append(fileList, File{
			Name:    info.Name(),
			IsDir:   info.IsDir(),
			Size:    FormatFileSize(info.Size()),
			ModTime: FormatModTime(info.ModTime().Format(time.RFC3339)),
			Path:    path.Join(reqPath, info.Name()), // relative path for client
		})
	}

	json.NewEncoder(w).Encode(fileList)
}
