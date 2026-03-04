package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var frontendFS embed.FS

// FrontendFS returns the embedded frontend filesystem.
func FrontendFS() (fs.FS, error) {
	return fs.Sub(frontendFS, "dist")
}
