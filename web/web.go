// Package web embeds the dashboard's static assets so they ship inside the
// binary — no files to copy into the container image.
package web

import "embed"

//go:embed static/*
var Files embed.FS
