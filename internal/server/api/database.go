// Eko: A terminal based social media platform
// Copyright (C) 2025 Kyren223
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package api

import (
	"database/sql"
	"embed"
	"log/slog"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"

	"github.com/kyren223/eko/pkg/assert"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

var db *sql.DB

func ConnectToDatabase() {
	var err error
	db, err = sql.Open("sqlite3", "file:server.db?cache=shared")
	if err != nil {
		slog.Error("unable to open database", "error", err)
		assert.Abort("see logs")
	}
	assert.AddFlush(db)
	slog.Info("established connection with the database")

	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA temp_store = MEMORY;",
		"PRAGMA mmap_size = 30000000000;",
	}
	for _, pragma := range pragmas {
		_, err := db.Exec(pragma)
		assert.NoError(err, "DB pragmas should always execute with no errors")
	}

	slog.Info("opened database, running migrations...")

	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		assert.NoError(err, "sqlite3 is a valid dialect that shouldn't error")
	}
	if err := goose.Up(db, "migrations"); err != nil {
		_ = db.Close()
		slog.Error("error running migrations", "error", err)
		assert.Abort("see logs")
	}

	slog.Info("database connection ready to be used")
}

func DB() *sql.DB {
	return db
}
