package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tachRoutine/beamdrop-go/pkg/db"
	"github.com/tachRoutine/beamdrop-go/pkg/logger"
)

type ShareableLinkHandler struct {
	sharedDir string
}

func NewShareableLinkHandler(sharedDir string) *ShareableLinkHandler {
	return &ShareableLinkHandler{sharedDir: sharedDir}
}

// CreateShareableLinkRequest represents the request to create a shareable link
type CreateShareableLinkRequest struct {
	Path      string `json:"path"`
	Password  string `json:"password,omitempty"`
	ExpiresIn *int64 `json:"expiresIn,omitempty"` // Duration in seconds
}

// CreateShareableLinkResponse represents the response after creating a shareable link
type CreateShareableLinkResponse struct {
	Token     string     `json:"token"`
	URL       string     `json:"url"`
	Path      string     `json:"path"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

// ShareableLinkInfo represents link information without sensitive data
type ShareableLinkInfo struct {
	ID          uint       `json:"id"`
	Path        string     `json:"path"`
	Token       string     `json:"token"`
	HasPassword bool       `json:"hasPassword"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	AccessCount int        `json:"accessCount"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// ValidateLinkRequest represents the request to validate a link with password
type ValidateLinkRequest struct {
	Password string `json:"password,omitempty"`
}

// Create handles POST requests to create a new shareable link
func (h *ShareableLinkHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateShareableLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		sendJSONError(w, "Path is required", http.StatusBadRequest)
		return
	}

	// Validate that the path exists
	fullPath := filepath.Join(h.sharedDir, req.Path)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		sendJSONError(w, "Path does not exist", http.StatusNotFound)
		return
	}

	var expiresIn *time.Duration
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		duration := time.Duration(*req.ExpiresIn) * time.Second
		expiresIn = &duration
	}

	link, err := db.CreateShareableLink(req.Path, req.Password, expiresIn)
	if err != nil {
		logger.Error("Failed to create shareable link: %v", err)
		sendJSONError(w, "Failed to create shareable link", http.StatusInternalServerError)
		return
	}

	// Construct the shareable URL
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	shareURL := fmt.Sprintf("%s://%s/share/%s", scheme, r.Host, link.Token)

	response := CreateShareableLinkResponse{
		Token:     link.Token,
		URL:       shareURL,
		Path:      link.Path,
		ExpiresAt: link.ExpiresAt,
		CreatedAt: link.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
	logger.Info("Shareable link created for path: %s", req.Path)
}

// List handles GET requests to list all shareable links
func (h *ShareableLinkHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	links, err := db.ListShareableLinks()
	if err != nil {
		logger.Error("Failed to list shareable links: %v", err)
		sendJSONError(w, "Failed to retrieve shareable links", http.StatusInternalServerError)
		return
	}

	// Convert to response format (hide sensitive data)
	var response []ShareableLinkInfo
	for _, link := range links {
		response = append(response, ShareableLinkInfo{
			ID:          link.ID,
			Path:        link.Path,
			Token:       link.Token,
			HasPassword: link.PasswordHash != "",
			ExpiresAt:   link.ExpiresAt,
			AccessCount: link.AccessCount,
			CreatedAt:   link.CreatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Delete handles DELETE requests to remove a shareable link
func (h *ShareableLinkHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		sendJSONError(w, "Token is required", http.StatusBadRequest)
		return
	}

	if err := db.DeleteShareableLink(token); err != nil {
		logger.Error("Failed to delete shareable link: %v", err)
		sendJSONError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Shareable link deleted successfully"})
	logger.Info("Shareable link deleted: %s", token)
}

// Access handles GET requests to access a file/folder via shareable link (public API endpoint)
func (h *ShareableLinkHandler) Access(w http.ResponseWriter, r *http.Request) {
	// Extract token from URL path: /api/shares/access/<token>
	token := strings.TrimPrefix(r.URL.Path, "/api/shares/access/")
	if token == "" {
		sendJSONError(w, "Invalid share URL", http.StatusBadRequest)
		return
	}

	// Get the shareable link
	link, err := db.GetShareableLinkByToken(token)
	if err != nil {
		logger.Error("Error accessing shareable link: %v", err)
		sendJSONError(w, "Invalid or expired link", http.StatusNotFound)
		return
	}
	if link == nil {
		sendJSONError(w, "Link not found", http.StatusNotFound)
		return
	}

	// Handle password-protected links
	if link.PasswordHash != "" {
		// Check if password is provided in query or via POST
		password := r.URL.Query().Get("password")

		// If no password in query, check request body for POST
		if password == "" && r.Method == http.MethodPost {
			var req ValidateLinkRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				password = req.Password
			}
		}

		if password == "" {
			// Return info that password is required
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"requiresPassword": true,
				"path":             link.Path,
			})
			return
		}

		if !db.ValidateLinkPassword(link, password) {
			sendJSONError(w, "Invalid password", http.StatusUnauthorized)
			return
		}
	}

	// Increment access count
	if err := db.IncrementAccessCount(token); err != nil {
		logger.Error("Failed to increment access count: %v", err)
	}

	// Serve the file or directory
	fullPath := filepath.Join(h.sharedDir, link.Path)

	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		logger.Error("Failed to stat path: %v", err)
		sendJSONError(w, "File or folder not found", http.StatusNotFound)
		return
	}

	if fileInfo.IsDir() {
		// For directories, return file listing
		files, err := os.ReadDir(fullPath)
		if err != nil {
			logger.Error("Failed to read directory: %v", err)
			sendJSONError(w, "Failed to read directory", http.StatusInternalServerError)
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
				Path:    filepath.Join(link.Path, info.Name()),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"path":  link.Path,
			"files": fileList,
			"isDir": true,
		})
	} else {
		// For files, serve the file for download
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(link.Path)))
		http.ServeFile(w, r, fullPath)
	}

	logger.Info("Shareable link accessed: %s for path: %s", token, link.Path)
}
