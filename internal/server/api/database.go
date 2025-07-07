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
