package data

import "embed"

//go:embed migrations/*.sql
var EmbeddedFiles embed.FS
