package database

import "embed"

//go:embed migrations/*.sql
var migrations embed.FS

//go:embed migrations_postgres/*.sql
var migrationsPostgres embed.FS
