package tg_provider

import (
	"aliorToDoBot/src/config"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type providerGroup interface {
	GetGroup(IDGroup int64) (string, error)
	CreateGroup(GroupName string) error
	DeleteGroup(GroupName string) error
}

type providerUser interface {
	GetUser(ChatID int64) (int64, string, error)
	CreateUser(ChatID int64, UserName string) error
}

type providerEvent interface {
	GetEvents(IDUser int64) (string, error)
	CreateEvent(GroupName string, NameEvent string, Category string,
		IsAllDay bool, DatetimeStart time.Time,
		Duration time.Duration) error
	DeleteEvent(NameEvent string) error
}

type DatabaseProvider interface {
	providerUser
	providerGroup
	providerEvent
}

var bot *tgbotapi.BotAPI

func GetUpdates() tgbotapi.UpdatesChannel {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	return bot.GetUpdatesChan(u)
}

// NewMyTgBot создает и возвращает новый экземпляр Telegram бота
func NewMyTgBot(cfg *config.TelegramConfig) (*MyTgBot, error) {
	var once sync.Once
	once.Do(func() {
		b, err := tgbotapi.NewBotAPI(cfg.Token)
		if err != nil {
			panic(err)
		}
		bot = b
	})
	return &MyTgBot{}, nil
}

type MyTgBot struct {
	database     DatabaseProvider
	chatID       int64
	lastActivity time.Time
}

func (m *MyTgBot) LastActivity() time.Duration {
	return time.Now().Sub(m.lastActivity)
}

func (m *MyTgBot) updateLastActivity() {
	m.lastActivity = time.Now()
}

func (m *MyTgBot) Send(messageText string) error {
	_, err := bot.Send(tgbotapi.NewMessage(m.chatID, messageText))
	m.updateLastActivity()
	return err
}

// StartUpdatesLoop запускает цикл обработки обновлений
func (m *MyTgBot) StartUpdatesLoop(handler func(update tgbotapi.Update)) {
	go func() {
		for update := range GetUpdates() {
			handler(update)
		}
	}()
}
