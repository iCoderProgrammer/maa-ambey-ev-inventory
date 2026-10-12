// Package seeds embeds the idempotent reference-data SQL files so cmd/seed is
// self-contained and never depends on the working directory.
package seeds

import "embed"

// FS holds every *.sql seed file in this directory.
//
//go:embed *.sql
var FS embed.FS
