package tg_provider

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// StartBot запускает основную логику работы бота
func StartBot(bot *tgbotapi.BotAPI) {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updatesChannel := bot.GetUpdatesChan(updateConfig)

	// Обрабатываем каждое обновление
	for updateMessage := range updatesChannel {
		if updateMessage.Message != nil { // Если это текстовое сообщение
			log.Printf("[%s] %s", updateMessage.Message.From.UserName, updateMessage.Message.Text)

			// Ответ пользователю
			msg := tgbotapi.NewMessage(updateMessage.Message.Chat.ID, "Привет! Ваше сообщение: "+updateMessage.Message.Text)
			_, err := bot.Send(msg)
			if err != nil {
				log.Printf("Ошибка отправки сообщения: %v", err)
			}

		}
	}
}
