package database

import "embed"

//go:embed migrations/*.sql
var migrations embed.FS
