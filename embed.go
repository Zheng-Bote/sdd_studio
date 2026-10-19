package sddstudio

import "embed"

// EmbeddedAssets contains the built-in rule catalog and handbook files so that
// the sdd binary operates standalone in any target repository without external files.
//
//go:embed catalog/catalog.yaml handbuch/*.md
var EmbeddedAssets embed.FS
