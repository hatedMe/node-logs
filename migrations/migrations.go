// Package migrations 内嵌 SQL 迁移文件，供 store.Migrate 按序应用。
package migrations

import "embed"

// FS 包含全部 *.sql 迁移文件。
//
//go:embed *.sql
var FS embed.FS
