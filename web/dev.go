//go:build dev

package web

import (
	"io/fs"
	"os"
)

// FrontendFSDev returns the frontend filesystem from disk for development.
func FrontendFSDev() fs.FS {
	return os.DirFS("frontend/dist")
}
