package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	`aliorToDoBot/src/db/migrations`

	_ "aliorToDoBot/migrations"
)

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://postgres:1q2w3e@localhost:5432/AliorToDoBot")
	if err != nil {
		log.Fatal(err)
	}
	err = migrations.MigrateDatabase(pool)
	if err != nil {
		log.Fatal(err)
	}
}
