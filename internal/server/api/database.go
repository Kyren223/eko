package api

import (
	"database/sql"
	"embed"
	"log"

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
		log.Fatalln("unable to open db:", err)
	}
	assert.AddFlush(db)
	log.Println("established connection with the database")

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

	log.Println("opened database, running up migrations...")

	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		assert.NoError(err, "sqlite3 is a valid dialect that shouldn't error")
	}
	if err := goose.Up(db, "migrations"); err != nil {
		db.Close()
		log.Fatalln("error running up migrations:", err)
	}
	log.Println("database up migrations applied successfully")

	log.Println("database connection ready to be used")
}

func DB() *sql.DB {
	return db
}
