package handlers

import (
	"net/http"

	"github.com/ekilie/beamdrop/static"
)

// LLMsTxtHandler serves the /llms.txt file for LLM discoverability.
func LLMsTxtHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	content := static.LLMsTxt
	if r.URL.Query().Get("full") == "true" {
		content = static.LLMsFullTxt
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(content))
}
