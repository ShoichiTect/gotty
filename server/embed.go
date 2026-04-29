package server

import "embed"

// StaticFS embeds the static assets directory at build time.
//
//go:embed static
var StaticFS embed.FS
