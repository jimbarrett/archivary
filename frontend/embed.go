package frontend

import "embed"

// Dist contains the built frontend assets from the dist/ directory.
// This is embedded at compile time so the binary is fully self-contained.
//
//go:embed dist/*
var Dist embed.FS
