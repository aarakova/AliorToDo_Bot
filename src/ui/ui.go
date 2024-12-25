package ui

import (
	"aliorToDoBot/src/config"
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"sync"
	"time"

	"aliorToDoBot/src/tg_provider"
)

// Bot интерфейс для работы с Telegram API
type Bot interface {
	Send(messageText string) error
	LastActivity() time.Duration
}

// NewUI создает новый объект UserInterface
func NewUI(cfg *config.UIConfig) *UserInterface {
	ui := &UserInterface{
		sessions: make(map[int64]*UserSession),
		ttl:      cfg.SessionTTL,
	}
	go ui.startCleaner(context.Background(), cfg.CleanerInterval*time.Minute)
	return ui
}

// UserInterface структура для работы с UI
type UserInterface struct {
	sessions map[int64]*UserSession
	mutex    sync.Mutex
	ttl      time.Duration
}

// UserSession структура для хранения данных сессии
type UserSession struct {
	step string
	Bot
}

// GetSession возвращает существующую или создает новую сессию для пользователя
func (u *UserInterface) GetSession(chatID int64) *UserSession {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	if session, exists := u.sessions[chatID]; exists {
		return session
	}
	us := &UserSession{
		Bot:  tg_provider.NewMyTgBot(chatID),
		step: "",
	}
	u.sessions[chatID] = us
	return us
}

// startCleaner запускает процесс очистки устаревших сессий
func (u *UserInterface) startCleaner(ctx context.Context, ticker *time.Ticker) {
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			u.clean()
		}
	}
}

// clean удаляет устаревшие сессии
func (u *UserInterface) clean() {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	for chatID, session := range u.sessions {
		if session.LastActivity() > u.ttl {
			delete(u.sessions, chatID)
		}
	}
}

// HandleUpdate Обработка входящих обновлений
func (u *UserInterface) HandleUpdate(update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		// Обработка callback-запросов
		u.handleCallbackQuery(update.CallbackQuery)
		return
	}

	if update.Message != nil {
		// Обработка сообщений
		chatID := update.Message.Chat.ID
		text := update.Message.Text
		userName := update.Message.Chat.UserName

		// Получаем текущий шаг пользователя
		userSession := u.GetSession(chatID)
		userStep := userSession.step

		switch userStep {
		case "creating_event_category", "creating_event_name", "creating_event_time", "creating_event_duration", "creating_event_all_day_date":
			u.handleEventCreation(chatID, text)
		case "creating_group_name", "adding_group_members":
			u.handleGroupCreation(chatID, text)
		default:
			u.handleDefault(chatID, text, userName)
		}
	}
}

// handleCallbackQuery обрабатывает callback-запросы
func (u *UserInterface) handleCallbackQuery(callbackQuery *tg_provider.CallbackQuery) {
	// Реализуйте логику обработки callback-запросов
}

// handleEventCreation обрабатывает создание события
func (u *UserInterface) handleEventCreation(chatID int64, text string) {
	// Реализуйте логику создания события
}

// handleGroupCreation обрабатывает создание группы
func (u *UserInterface) handleGroupCreation(chatID int64, text string) {
	// Реализуйте логику создания группы
}

// handleDefault обрабатывает действия по умолчанию
func (u *UserInterface) handleDefault(chatID int64, text, userName string) {
	u.bot.Send(chatID, "Привет, "+userName+"! Ваше сообщение: "+text)
}
