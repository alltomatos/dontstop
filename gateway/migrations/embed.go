package migrations

import "embed"

// FS contains the versioned SQL migrations embedded in the gateway binary.
//
//go:embed *.sql
var FS embed.FS
