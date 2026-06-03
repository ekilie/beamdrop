package beam

import "time"

type ServerStats struct{
	Downloads int       `json:"downloads"`
	Requests  int       `json:"requests"`
    Uploads   int       `json:"uploads"`
    StartTime time.Time `json:"startTime"`
}

type File struct {
	Name    string `json:"name"`
	Size    string `json:"size"`
	IsDir   bool   `json:"isDir"`
	ModTime string `json:"modTime"`
	Path    string `json:"path"`
}