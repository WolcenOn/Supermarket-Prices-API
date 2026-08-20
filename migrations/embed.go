package migrations

import "embed"

// Files contains all SQL migrations so the migration binary can run without
// relying on repository files being present in the runtime container.
//
//go:embed *.sql
var Files embed.FS
