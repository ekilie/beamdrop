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
	//TODO: im gonna do something here later not sure what yet
}
