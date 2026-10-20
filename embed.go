package sddstudio

import "embed"

// EmbeddedAssets contains the built-in rule catalog, handbook and artifact templates so that
// the sdd binary operates standalone in any target repository without external files.
//
//go:embed catalog/catalog.yaml handbuch/*.md templates/*.tmpl
var EmbeddedAssets embed.FS
