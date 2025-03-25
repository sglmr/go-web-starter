package assets

import (
	"embed"
)

//go:embed "static" "templates" "emails"
var EmbeddedFiles embed.FS
