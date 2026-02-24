package handlers

import (
    "log/slog"
    "net/http"
    "path/filepath"
    "strconv"
    "strings"

    "github.com/ekilie/beamdrop/pkg/db"
    "github.com/ekilie/beamdrop/pkg/storage"
)

type DownloadHandler struct {
    objectManager *storage.ObjectManager
}

func NewDownloadHandler(sharedDir string) *DownloadHandler {
    return &DownloadHandler{
        objectManager: storage.NewObjectManager(sharedDir),
    }
}

func (h *DownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Extract token from /dl/{token}
    token := strings.TrimPrefix(r.URL.Path, "/dl/")
    if token == "" {
        sendJSONError(w, "Missing token", http.StatusBadRequest)
        return
    }

    // Look up the presigned URL
    p, err := db.GetPresignedURLByToken(token)
    if err != nil {
        slog.Debug("Presigned URL access denied", "token", token, "error", err)
        sendJSONError(w, "Invalid or expired link", http.StatusGone)
        return
    }
    if p == nil {
        sendJSONError(w, "Link not found", http.StatusNotFound)
        return
    }

    // Only allow GET downloads via this endpoint
    if p.Method != "GET" {
        sendJSONError(w, "This link does not allow downloads", http.StatusForbidden)
        return
    }

    // Get the file from storage
    file, info, unlock, err := h.objectManager.GetObject(p.Bucket, p.Key)
    if err != nil {
        slog.Error("Failed to get object for presigned URL", "bucket", p.Bucket, "key", p.Key, "error", err)
        sendJSONError(w, "File not found", http.StatusNotFound)
        return
    }
    defer file.Close()
    defer unlock()

    // Increment download count
    if err := db.IncrementPresignedURLDownloads(token); err != nil {
        slog.Error("Failed to increment download count", "token", token, "error", err)
        // Non-fatal — continue serving the file
    }

    // Set response headers
    w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
    w.Header().Set("Last-Modified", info.LastModified.UTC().Format(http.TimeFormat))

    ext := strings.ToLower(filepath.Ext(p.Key))
    contentType := getContentTypeForExt(ext)
    w.Header().Set("Content-Type", contentType)

    // Use http.ServeContent for Range support
    http.ServeContent(w, r, p.Key, info.LastModified, file)
}

// getContentTypeForExt returns a content type for a file extension.
// Duplicates the logic from api/objects.go — consider extracting to a shared helper.
func getContentTypeForExt(ext string) string {
    types := map[string]string{
        ".html": "text/html",
        ".css":  "text/css",
        ".js":   "application/javascript",
        ".json": "application/json",
        ".png":  "image/png",
        ".jpg":  "image/jpeg",
        ".jpeg": "image/jpeg",
        ".gif":  "image/gif",
        ".svg":  "image/svg+xml",
        ".pdf":  "application/pdf",
        ".zip":  "application/zip",
        ".txt":  "text/plain",
        ".xml":  "application/xml",
        ".webp": "image/webp",
        ".mp4":  "video/mp4",
        ".mp3":  "audio/mpeg",
    }
    if ct, ok := types[ext]; ok {
        return ct
    }
    return "application/octet-stream"
}