package static

import (
	"embed"
)

//go:embed all:frontend/dist
var FrontendFiles embed.FS

//go:embed llms.txt
var LLMsTxt string

//go:embed llms-full.txt
var LLMsFullTxt string

func init() {
	// Verify the frontend build was embedded. Without this, forgetting to
	// build the frontend before building the Go binary only surfaces as a
	// runtime 404.
	_, err := FrontendFiles.Open("frontend/dist/index.html")
	if err != nil {
		panic("static: frontend build not embedded. Run `cd static/frontend && pnpm run build` before building the Go binary.")
	}
}

