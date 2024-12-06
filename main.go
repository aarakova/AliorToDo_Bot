package main

import (
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Event struct {
	ID        int
	Category  string
	Name      string
	StartTime time.Time
	Duration  time.Duration
	IsAllDay  bool
	Status    string
}

type Group struct {
	ID      int
	Name    string
	Members []string
}

var (
	events          = make(map[int]Event) // Временное хранилище мероприятий
	groups          = make(map[int]Group) // Временное хранилище групп
	eventCounter    = 1                   // Счетчик ID мероприятий
	groupCounter    = 1                   // Счетчик ID групп
	userSteps       = make(map[int64]string)
	tempEvent       = make(map[int64]Event) // Временное хранилище для событий на этапе создания
	tempGroup       = make(map[int64]Group) // Временное хранилище для групп на этапе создания
	authorizedUsers = map[string]int64{     // Мапа авторизованных пользователей: username -> chatID
		"@EgorKo25": 1233580695,
		"@aarachok": 917952137,
		"@deaqs":    182062937,
	}
)

func main() {
	bot, err := tgbotapi.NewBotAPI("7232931230:AAGsWxc4no6O1hPDAbgGLQcdb6ZLuCfmYgs")
	if err != nil {
		log.Fatalf("Ошибка подключения к Telegram API: %v", err)
	}

	bot.Debug = true
	log.Printf("Авторизован как %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		userStep := userSteps[chatID]

		switch userStep {
		case "creating_event_category", "creating_event_name", "creating_event_time", "creating_event_duration", "creating_event_all_day_date":
			handleEventCreation(bot, chatID, update.Message.Text)
		case "creating_group_name", "adding_group_members":
			handleGroupCreation(bot, chatID, update.Message.Text)
		default:
			handleDefault(bot, chatID, update.Message.Text)
		}
	}
}

// ---- Общие функции ----
func handleDefault(bot *tgbotapi.BotAPI, chatID int64, text string) {
	switch text {
	case "/start", "Главное меню":
		sendMainMenu(bot, chatID)
	case "Мероприятия":
		sendEventsMenu(bot, chatID)
	case "Группы":
		sendGroupsMenu(bot, chatID)
	case "Создать мероприятие":
		startCreateEvent(bot, chatID)
	case "Создать группу":
		startCreateGroup(bot, chatID)
	case "Мои мероприятия":
		viewMyEvents(bot, chatID)
	case "Мои группы":
		viewMyGroups(bot, chatID)
	default:
		msg := tgbotapi.NewMessage(chatID, "Неизвестная команда. Используйте /start.")
		bot.Send(msg)
	}
}

func sendMainMenu(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Главное меню:")
	msg.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
		Keyboard: [][]tgbotapi.KeyboardButton{
			{tgbotapi.NewKeyboardButton("Мероприятия"), tgbotapi.NewKeyboardButton("Группы")},
		},
		ResizeKeyboard: true,
	}
	bot.Send(msg)
}

func sendEventsMenu(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Меню мероприятий:")
	msg.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
		Keyboard: [][]tgbotapi.KeyboardButton{
			{tgbotapi.NewKeyboardButton("Создать мероприятие"), tgbotapi.NewKeyboardButton("Мои мероприятия")},
			{tgbotapi.NewKeyboardButton("Главное меню")},
		},
		ResizeKeyboard: true,
	}
	bot.Send(msg)
}

func sendGroupsMenu(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Меню групп:")
	msg.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
		Keyboard: [][]tgbotapi.KeyboardButton{
			{tgbotapi.NewKeyboardButton("Создать группу"), tgbotapi.NewKeyboardButton("Мои группы")},
			{tgbotapi.NewKeyboardButton("Главное меню")},
		},
		ResizeKeyboard: true,
	}
	bot.Send(msg)
}

// ---- Функционал просмотра мероприятий ----
func viewMyEvents(bot *tgbotapi.BotAPI, chatID int64) {
	if len(events) == 0 {
		msg := tgbotapi.NewMessage(chatID, "У вас пока нет мероприятий.")
		bot.Send(msg)
		return
	}

	var message strings.Builder
	message.WriteString("Ваши мероприятия:\n")
	for _, event := range events {
		eventInfo := formatEvent(event)
		message.WriteString(eventInfo + "\n\n")
	}

	msg := tgbotapi.NewMessage(chatID, message.String())
	bot.Send(msg)
}

func formatEvent(event Event) string {
	if event.IsAllDay {
		return "Мероприятие: " + event.Name + "\nКатегория: " + event.Category + "\nДата: " + event.StartTime.Format("02.01.2006") + "\nСтатус: " + event.Status
	}
	return "Мероприятие: " + event.Name + "\nКатегория: " + event.Category + "\nДата и время: " + event.StartTime.Format("02.01.2006 15:04") + "\nПродолжительность: " + event.Duration.String() + "\nСтатус: " + event.Status
}

// ---- Функционал просмотра групп ----
func viewMyGroups(bot *tgbotapi.BotAPI, chatID int64) {
	var userGroups []Group
	for _, group := range groups {
		for _, member := range group.Members {
			if authorizedUsers[member] == chatID {
				userGroups = append(userGroups, group)
				break
			}
		}
	}

	if len(userGroups) == 0 {
		msg := tgbotapi.NewMessage(chatID, "У вас пока нет групп.")
		bot.Send(msg)
		return
	}

	var message strings.Builder
	message.WriteString("Ваши группы:\n")
	for _, group := range userGroups {
		message.WriteString("Группа: " + group.Name + "\nУчастники: " + strings.Join(group.Members, ", ") + "\n\n")
	}

	msg := tgbotapi.NewMessage(chatID, message.String())
	bot.Send(msg)
}

// ---- Функционал создания мероприятий ----
func startCreateEvent(bot *tgbotapi.BotAPI, chatID int64) {
	userSteps[chatID] = "creating_event_category"
	tempEvent[chatID] = Event{}

	msg := tgbotapi.NewMessage(chatID, "Выберите категорию мероприятия:")
	msg.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
		Keyboard: [][]tgbotapi.KeyboardButton{
			{tgbotapi.NewKeyboardButton("Личное"), tgbotapi.NewKeyboardButton("Семья"), tgbotapi.NewKeyboardButton("Работа")},
			{tgbotapi.NewKeyboardButton("Главное меню")},
		},
		ResizeKeyboard: true,
	}
	bot.Send(msg)
}

func handleEventCreation(bot *tgbotapi.BotAPI, chatID int64, text string) {
	event := tempEvent[chatID]

	switch userSteps[chatID] {
	case "creating_event_category":
		event.Category = text
		tempEvent[chatID] = event
		userSteps[chatID] = "creating_event_name"
		msg := tgbotapi.NewMessage(chatID, "Введите название мероприятия:")
		bot.Send(msg)
	case "creating_event_name":
		event.Name = text
		tempEvent[chatID] = event
		userSteps[chatID] = "creating_event_time"
		msg := tgbotapi.NewMessage(chatID, "Введите дату и время начала в формате дд.мм.гггг чч:мм или нажмите 'Весь день':")
		msg.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
			Keyboard: [][]tgbotapi.KeyboardButton{
				{tgbotapi.NewKeyboardButton("Весь день"), tgbotapi.NewKeyboardButton("Главное меню")},
			},
			ResizeKeyboard: true,
		}
		bot.Send(msg)
	case "creating_event_time":
		if strings.HasPrefix(text, "Весь день") {
			userSteps[chatID] = "creating_event_all_day_date"
			msg := tgbotapi.NewMessage(chatID, "Введите дату для мероприятия в формате дд.мм.гггг:")
			bot.Send(msg)
			return
		}
		layout := "02.01.2006 15:04"
		startTime, err := time.Parse(layout, text)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Неверный формат. Пожалуйста, введите дату и время в формате дд.мм.гггг чч:мм.")
			bot.Send(msg)
			return
		}
		event.StartTime = startTime
		tempEvent[chatID] = event
		userSteps[chatID] = "creating_event_duration"
		msg := tgbotapi.NewMessage(chatID, "Введите продолжительность мероприятия (например, 1d2h) или нажмите 'Пропустить':")
		msg.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
			Keyboard: [][]tgbotapi.KeyboardButton{
				{tgbotapi.NewKeyboardButton("Пропустить"), tgbotapi.NewKeyboardButton("Главное меню")},
			},
			ResizeKeyboard: true,
		}
		bot.Send(msg)
	case "creating_event_all_day_date":
		layout := "02.01.2006"
		allDayDate, err := time.Parse(layout, text)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Неверный формат. Пожалуйста, введите дату в формате дд.мм.гггг.")
			bot.Send(msg)
			return
		}
		event.StartTime = allDayDate
		event.IsAllDay = true
		tempEvent[chatID] = event
		userSteps[chatID] = "creating_event_duration"
		msg := tgbotapi.NewMessage(chatID, "Введите продолжительность мероприятия (например, 1d2h) или нажмите 'Пропустить':")
		msg.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
			Keyboard: [][]tgbotapi.KeyboardButton{
				{tgbotapi.NewKeyboardButton("Пропустить"), tgbotapi.NewKeyboardButton("Главное меню")},
			},
			ResizeKeyboard: true,
		}
		bot.Send(msg)
	case "creating_event_duration":
		if text != "Пропустить" {
			// Парсим продолжительность
			duration, err := parseDuration(text)
			if err != nil {
				msg := tgbotapi.NewMessage(chatID, "Неверный формат продолжительности. Используйте формат 1d2h3m.")
				bot.Send(msg)
				return
			}
			event.Duration = duration // Сохраняем продолжительность в объект event
		}
		// Завершаем создание мероприятия
		event.ID = eventCounter
		event.Status = "Запланировано"
		events[eventCounter] = event
		eventCounter++

		// Очистка временных данных
		delete(tempEvent, chatID)
		delete(userSteps, chatID)

		// Уведомление об успешном создании
		msg := tgbotapi.NewMessage(chatID, "Мероприятие успешно создано!")
		bot.Send(msg)
		sendMainMenu(bot, chatID)
	}
}

// ---- Функционал создания группы ----
func startCreateGroup(bot *tgbotapi.BotAPI, chatID int64) {
	userSteps[chatID] = "creating_group_name"
	tempGroup[chatID] = Group{}

	msg := tgbotapi.NewMessage(chatID, "Введите название группы:")
	msg.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
		Keyboard: [][]tgbotapi.KeyboardButton{
			{tgbotapi.NewKeyboardButton("Главное меню")},
		},
		ResizeKeyboard: true,
	}
	bot.Send(msg)
}

func handleGroupCreation(bot *tgbotapi.BotAPI, chatID int64, text string) {
	group := tempGroup[chatID]

	switch userSteps[chatID] {
	case "creating_group_name":
		// Устанавливаем название группы и добавляем создателя группы как участника
		group.Name = text
		creatorUsername := findUsernameByChatID(chatID)
		if creatorUsername != "" {
			group.Members = append(group.Members, creatorUsername) // Добавляем создателя группы
		}
		tempGroup[chatID] = group
		userSteps[chatID] = "adding_group_members"

		msg := tgbotapi.NewMessage(chatID, "Введите теги участников группы (через запятую):")
		bot.Send(msg)
	case "adding_group_members":
		// Разделяем введенные теги участников и добавляем их в группу
		members := strings.Split(text, ",")
		for _, member := range members {
			member = strings.TrimSpace(member)
			if member != "" && !contains(group.Members, member) {
				group.Members = append(group.Members, member)
			}
		}

		// Сохраняем группу в общий список
		group.ID = groupCounter
		groups[groupCounter] = group
		groupCounter++

		// Уведомляем всех участников (кроме создателя)
		for _, member := range group.Members {
			participantChatID, ok := authorizedUsers[member]
			if ok && participantChatID != chatID {
				msg := tgbotapi.NewMessage(participantChatID, "Вы добавлены в группу: "+group.Name)
				bot.Send(msg)
			}
		}

		// Завершаем создание группы
		delete(tempGroup, chatID)
		delete(userSteps, chatID)

		msg := tgbotapi.NewMessage(chatID, "Группа успешно создана!")
		bot.Send(msg)
		sendMainMenu(bot, chatID)
	}
}

// Вспомогательная функция для проверки наличия элемента в слайсе
func contains(slice []string, item string) bool {
	for _, elem := range slice {
		if elem == item {
			return true
		}
	}
	return false
}

// Вспомогательная функция для нахождения username по chatID
func findUsernameByChatID(chatID int64) string {
	for username, id := range authorizedUsers {
		if id == chatID {
			return username
		}
	}
	return ""
}

func parseDuration(input string) (time.Duration, error) {
	var totalDuration time.Duration
	var currentValue string

	// Разбираем строку символ за символом
	for _, char := range input {
		if char >= '0' && char <= '9' { // Если символ — цифра, добавляем к текущему значению
			currentValue += string(char)
		} else { // Если символ — суффикс (d, h, m, s)
			if currentValue == "" {
				return 0, nil // Если перед суффиксом не было числа, возвращаем ошибку
			}
			value, err := strconv.Atoi(currentValue)
			if err != nil {
				return 0, err
			}
			switch char {
			case 'd':
				totalDuration += time.Hour * 24 * time.Duration(value)
			case 'h':
				totalDuration += time.Hour * time.Duration(value)
			case 'm':
				totalDuration += time.Minute * time.Duration(value)
			case 's':
				totalDuration += time.Second * time.Duration(value)
			default:
				return 0, nil // Некорректный суффикс
			}
			currentValue = "" // Сбрасываем текущее значение после обработки
		}
	}

	// Если строка закончилась без суффикса, возвращаем ошибку
	if currentValue != "" {
		return 0, nil
	}

	return totalDuration, nil
}
