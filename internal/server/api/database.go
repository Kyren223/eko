package api

import (
	"database/sql"
	"log"

	"github.com/kyren223/eko/pkg/assert"
)

var db *sql.DB

func ConnectToDatabase() {
	db, err := sql.Open("sqlite3", "server.db")
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

func CloseDatabase() {
	db.Close()
	log.Println("connection with database closed")
}
