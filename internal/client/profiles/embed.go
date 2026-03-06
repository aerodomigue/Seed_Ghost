package profiles

import "embed"

//go:embed *.json
var BundledFS embed.FS
