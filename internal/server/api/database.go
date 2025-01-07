package api

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"

	"github.com/kyren223/eko/pkg/assert"
)

var db *sql.DB

func ConnectToDatabase() {
	var err error
	db, err = sql.Open("sqlite3", "file:server.db?cache=shared")
	assert.NoError(err, "DB should always be accessible")
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

	log.Println("database connection ready to be used")
}

func DB() *sql.DB {
	return db
}
