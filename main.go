package main

import (
	"context"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"aliorToDoBot/src/db/migrations"
	"aliorToDoBot/src/tg_provider"

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

	bot, err := tgbotapi.NewBotAPI("7232931230:AAGsWxc4no6O1hPDAbgGLQcdb6ZLuCfmYgs")

	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	tg_provider.StartBot(bot)
}
