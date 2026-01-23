package kdb

import (
	_ "embed"
)

//go:embed schema.sql
var Schema string

func SchemaSQL() string { return Schema }
