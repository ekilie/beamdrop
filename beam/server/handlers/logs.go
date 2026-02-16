package handlers

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/tachRoutine/beamdrop-go/pkg/errors"
	"github.com/tachRoutine/beamdrop-go/pkg/logger"
)

// LogEntry represents a single structured JSON log line.
type LogEntry struct {
	Time    string         `json:"time"`
	Level   string         `json:"level"`
	Msg     string         `json:"msg"`
	Source  map[string]any `json:"source,omitempty"`
	Attrs   map[string]any `json:"attrs,omitempty"` // extra fields
	RawJSON string         `json:"-"`
}

// LogsResponse is the JSON envelope returned by the /logs endpoint.
type LogsResponse struct {
	Logs       []map[string]any `json:"logs"`
	Total      int              `json:"total"`
	Returned   int              `json:"returned"`
	HasMore    bool             `json:"hasMore"`
	LogPath    string           `json:"logPath"`
}

// LogsHandler returns an HTTP handler that reads and serves structured JSON
// logs from the beamdrop log file.
//
// Query parameters:
//   - limit:  max entries to return (default 200, max 5000)
//   - offset: number of entries to skip from the end (for pagination)
//   - level:  filter by log level (debug, info, warn, error)
//   - search: case-insensitive substring match on the message field
func LogsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			errors.New(errors.CodeInvalidRequest, errors.CategoryValidation,
				"Method not allowed", http.StatusMethodNotAllowed).WriteHTTPResponse(w)
			return
		}

		logPath := logger.LogPath()
		if logPath == "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(LogsResponse{
				Logs:    make([]map[string]any, 0),
				Total:   0,
				LogPath: "",
			})
			return
		}

		// Parse query params
		limit := 200
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 {
				limit = n
			}
		}
		if limit > 5000 {
			limit = 5000
		}

		offset := 0
		if o := r.URL.Query().Get("offset"); o != "" {
			if n, err := strconv.Atoi(o); err == nil && n >= 0 {
				offset = n
			}
		}

		levelFilter := strings.ToUpper(r.URL.Query().Get("level"))
		searchFilter := strings.ToLower(r.URL.Query().Get("search"))

		// Read and parse log file
		allEntries, err := readLogEntries(logPath)
		if err != nil {
			slog.Error("Failed to read log file", "error", err)
			errors.InternalError("Failed to read log file").WithCause(err).WriteHTTPResponse(w)
			return
		}

		// Apply filters
		var filtered []map[string]any
		for _, entry := range allEntries {
			// Level filter
			if levelFilter != "" {
				entryLevel, _ := entry["level"].(string)
				if !strings.EqualFold(entryLevel, levelFilter) {
					continue
				}
			}

			// Search filter (case-insensitive match on msg)
			if searchFilter != "" {
				msg, _ := entry["msg"].(string)
				if !strings.Contains(strings.ToLower(msg), searchFilter) {
					continue
				}
			}

			filtered = append(filtered, entry)
		}

		total := len(filtered)

		// Apply offset and limit (entries are newest-first)
		start := offset
		if start > total {
			start = total
		}
		end := start + limit
		if end > total {
			end = total
		}

		page := filtered[start:end]
		if page == nil {
			page = make([]map[string]any, 0)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LogsResponse{
			Logs:     page,
			Total:    total,
			Returned: len(page),
			HasMore:  end < total,
			LogPath:  logPath,
		})
	}
}

// readLogEntries reads the JSON log file and returns entries newest-first.
func readLogEntries(logPath string) ([]map[string]any, error) {
	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []map[string]any
	scanner := bufio.NewScanner(f)
	// Increase buffer size for potentially long log lines (1MB)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // skip malformed lines
		}
		entries = append(entries, entry)
	}

	// Reverse to newest-first
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries, scanner.Err()
}
