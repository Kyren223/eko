package api

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"

	"github.com/kyren223/eko/internal/data"
	"github.com/kyren223/eko/pkg/assert"
	"github.com/kyren223/eko/pkg/snowflake"
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

func demo() {
	ctx := context.Background()
	node := snowflake.NewNode(1)
	pubKey, _, _ := ed25519.GenerateKey(nil)

	queries := data.New(db)
	user, _ := queries.CreateUser(ctx, data.CreateUserParams{
		ID:        node.Generate(),
		PublicKey: pubKey,
	})
	user, _ = queries.SetUserName(ctx, data.SetUserNameParams{
		ID:   user.ID,
		Name: "admin",
	})
	network, _ := queries.CreateNetwork(ctx, data.CreateNetworkParams{
		ID:      node.Generate(),
		Name:    "global",
		OwnerID: user.ID,
	})
	_, _ = queries.CreateFrequency(ctx, data.CreateFrequencyParams{
		ID:        node.Generate(),
		NetworkID: network.ID,
		Name:      "general",
	})
}
