package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"

	"aliorToDoBot/gorm_models"
	"aliorToDoBot/src/db"
)

// type Event struct {
// 	ID        int64
// 	GroupID   int64    // ID группы, если мероприятие связано с группой
// 	Category  string // Личное, Семья, Работа
// 	Name      string
// 	StartTime time.Time
// 	Duration  time.Duration
// 	IsAllDay  bool
// 	Status    string
// }

// type Group struct {
// 	ID      int
// 	Name    string
// 	Members []string
// }

var (
	// events          = make(map[int]Event) // Временное хранилище мероприятий
	// groups          = make(map[int]Group) // Временное хранилище групп
	// eventCounter    = 1                   // Счетчик ID мероприятий
	// groupCounter    = 1                   // Счетчик ID групп
	userSteps       = make(map[int64]string)
	tempEvent       = make(map[int64]gorm_models.Event) // Временное хранилище для событий на этапе создания
	tempGroup       = make(map[int64]gorm_models.Group) // Временное хранилище для групп на этапе создания
	authorizedUsers = map[string]int64{                 // Мапа авторизованных пользователей: username -> chatID
		"@EgorKo25": 1233580695,
		"@aarachok": 917952137,
		"@deaqs":    182062937,
	}
)

func main() {

	// Строка подключения к PostgreSQL
	dsn := "host=localhost user=postgres password=password dbname=AliorToDoBot port=5432 sslmode=disable"

	// Инициализация базы данных через GORM
	db.InitGormDatabase(dsn)

	// Автоматическая миграция моделей
	err := db.DB.AutoMigrate(
		&gorm_models.User{},
		&gorm_models.Group{},
		&gorm_models.Event{},
		&gorm_models.Membership{},
	)
	if err != nil {
		log.Fatalf("Ошибка миграции: %v", err)
	}

	log.Println("База данных успешно инициализирована и обновлена!")

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
		if update.CallbackQuery != nil {
			// Обработка callback-запросов
			handleCallbackQuery(bot, update.CallbackQuery)
			continue
		}

		if update.Message != nil {
			// Обработка сообщений
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
}

// ---- Общие функции ----
func handleDefault(bot *tgbotapi.BotAPI, chatID int64, text string) {
	if text == "Главное меню" {
		delete(userSteps, chatID)
		sendMainMenu(bot, chatID)
		return
	}

	switch text {
	case "/start":
		ensurePersonalGroup(bot, chatID)
		sendMainMenu(bot, chatID)
	case "Мероприятия":
		sendEventsMenu(bot, chatID)
	case "Группы":
		sendGroupsMenu(bot, chatID)
	case "Создать мероприятие":
		startCreateEvent(bot, chatID)
	// case "Создать мероприятие для группы":
	// 	startGroupEventCreation(bot, chatID)
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
func ensurePersonalGroup(bot *tgbotapi.BotAPI, chatID int64) {
	// Проверяем, есть ли уже группа "Личное" для данного пользователя
	var group gorm_models.Group
	err := db.DB.Where("group_name = ? AND id_group IN (SELECT id_group FROM memberships WHERE id_user = ?)", "Личное", chatID).First(&group).Error

	// Если произошла ошибка, но она не связана с отсутствием записи, логируем и выходим
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Ошибка проверки группы 'Личное' для пользователя %d: %v", chatID, err)
		return
	}

	// Если группа "Личное" уже существует, выходим
	if err == nil {
		log.Printf("Группа 'Личное' уже существует для пользователя %d", chatID)
		return
	}

	// Создаем группу "Личное"
	newGroup := gorm_models.Group{
		GroupName: "Личное",
	}

	if err := db.DB.Create(&newGroup).Error; err != nil {
		log.Printf("Ошибка создания группы 'Личное' для пользователя %d: %v", chatID, err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка при создании группы 'Личное'. Попробуйте позже.")
		bot.Send(msg)
		return
	}

	// Добавляем запись о членстве (Membership) для администратора
	membership := gorm_models.Membership{
		IDGroup: newGroup.IDGroup,
		IDUser:  chatID,
		IDAdmin: chatID, // Устанавливаем текущего пользователя администратором
	}

	if err := db.DB.Create(&membership).Error; err != nil {
		log.Printf("Ошибка добавления пользователя %d в группу 'Личное': %v", chatID, err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка при создании группы 'Личное'. Попробуйте позже.")
		bot.Send(msg)
		return
	}

	log.Printf("Группа 'Личное' успешно создана для пользователя %d", chatID)
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
			{tgbotapi.NewKeyboardButton("Создать мероприятие")},
			{tgbotapi.NewKeyboardButton("Главное меню"), tgbotapi.NewKeyboardButton("Мои мероприятия")},
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
	// Найти группы, в которых пользователь состоит
	var memberships []gorm_models.Membership
	if err := db.DB.Where("id_user = ?", chatID).Find(&memberships).Error; err != nil {
		log.Println("Ошибка получения групп пользователя:", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка при получении ваших групп.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
		return
	}

	if len(memberships) == 0 {
		msg := tgbotapi.NewMessage(chatID, "У вас пока нет мероприятий.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
		return
	}

	// Собираем ID групп
	groupIDs := make([]int64, 0)
	for _, membership := range memberships {
		groupIDs = append(groupIDs, membership.IDGroup)
	}

	// Находим мероприятия, связанные с этими группами
	var events []gorm_models.Event
	if err := db.DB.Where("id_group IN ?", groupIDs).Find(&events).Error; err != nil {
		log.Println("Ошибка получения event записей:", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка при получении ваших мероприятий.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
		return
	}

	if len(events) == 0 {
		msg := tgbotapi.NewMessage(chatID, "У вас пока нет мероприятий.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
		return
	}

	// Извлекаем данные о группах для отображения
	var groups []gorm_models.Group
	if err := db.DB.Where("id_group IN ?", groupIDs).Find(&groups).Error; err != nil {
		log.Println("Ошибка получения данных групп:", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка при обработке данных ваших групп.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
		return
	}

	groupMap := make(map[int64]string)
	for _, group := range groups {
		groupMap[group.IDGroup] = group.GroupName
	}

	// Формируем список мероприятий для отображения
	var message strings.Builder
	message.WriteString("Ваши мероприятия:\n\n")
	for _, event := range events {
		groupName := groupMap[event.IDGroup]
		message.WriteString(formatEvent(event, groupName) + "\n")
	}

	msg := tgbotapi.NewMessage(chatID, message.String())
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

func formatEvent(event gorm_models.Event, groupName string) string {
	if event.IsAllDay {
		return fmt.Sprintf("📅 *%s*\nГруппа: %s\nКатегория: %s\nДата: %s\nСтатус: %s",
			event.NameEvent, groupName, event.Category, event.DatetimeStart.Format("02.01.2006"), event.Status)
	}
	return fmt.Sprintf("📅 *%s*\nГруппа: %s\nКатегория: %s\nДата и время: %s\nПродолжительность: %s\nСтатус: %s",
		event.NameEvent, groupName, event.Category, event.DatetimeStart.Format("02.01.2006 15:04"), event.Duration.String(), event.Status)
}

// ---- Функционал просмотра групп ----
func viewMyGroups(bot *tgbotapi.BotAPI, chatID int64) {
	var memberships []gorm_models.Membership
	if err := db.DB.Where("id_user = ?", chatID).Find(&memberships).Error; err != nil {
		log.Println("Ошибка получения membership записей:", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка при получении ваших групп.")
		bot.Send(msg)
		return
	}

	if len(memberships) == 0 {
		msg := tgbotapi.NewMessage(chatID, "У вас пока нет групп.")
		bot.Send(msg)
		return
	}

	groupIDs := make([]int64, 0)
	for _, membership := range memberships {
		groupIDs = append(groupIDs, membership.IDGroup)
	}

	var groups []gorm_models.Group
	if err := db.DB.Where("id_group IN (?)", groupIDs).Find(&groups).Error; err != nil {
		log.Println("Ошибка получения групп:", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка при получении ваших групп.")
		bot.Send(msg)
		return
	}

	var message strings.Builder
	message.WriteString("Ваши группы:\n\n")
	for _, group := range groups {
		var groupMemberships []gorm_models.Membership
		if err := db.DB.Where("id_group = ?", group.IDGroup).Find(&groupMemberships).Error; err != nil {
			log.Printf("Ошибка получения участников для группы %d: %v", group.IDGroup, err)
			continue
		}

		members := make([]string, 0)
		admin := ""
		for _, membership := range groupMemberships {
			var user gorm_models.User
			if err := db.DB.Where("id_user = ?", membership.IDUser).First(&user).Error; err == nil {
				if membership.IDAdmin == membership.IDUser {
					admin = user.UserName
				} else {
					members = append(members, user.UserName)
				}
			}
		}
		message.WriteString(fmt.Sprintf("Группа: %s\nАдминистратор: %s\nУчастники: %s\n\n", group.GroupName, admin, strings.Join(members, ", ")))
	}

	msg := tgbotapi.NewMessage(chatID, message.String())
	bot.Send(msg)
}

// ---- Функционал создания мероприятий ----
func startCreateEvent(bot *tgbotapi.BotAPI, chatID int64) {
	// Получаем список групп, где пользователь является администратором
	var memberships []gorm_models.Membership
	err := db.DB.Where("id_user = ? AND id_admin = ?", chatID, chatID).Find(&memberships).Error
	if err != nil {
		log.Println("Ошибка получения групп пользователя:", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка при получении ваших групп.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
		return
	}

	// Собираем ID групп
	groupIDs := make([]int64, 0)
	for _, membership := range memberships {
		groupIDs = append(groupIDs, membership.IDGroup)
	}

	// Извлекаем данные о группах
	var groups []gorm_models.Group
	if len(groupIDs) > 0 {
		if err := db.DB.Where("id_group IN ?", groupIDs).Find(&groups).Error; err != nil {
			log.Println("Ошибка получения информации о группах:", err)
			msg := tgbotapi.NewMessage(chatID, "Ошибка при получении данных о группах.")
			if _, err := bot.Send(msg); err != nil {
				log.Printf("Ошибка отправки сообщения: %v", err)
			}
			return
		}
	}

	if len(groups) == 0 {
		msg := tgbotapi.NewMessage(chatID, "У вас нет групп, в которых вы являетесь администратором.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
		return
	}

	// Создаем инлайн-кнопки для выбора группы
	var inlineKeyboard [][]tgbotapi.InlineKeyboardButton
	for _, group := range groups {
		button := tgbotapi.NewInlineKeyboardButtonData(group.GroupName, fmt.Sprintf("group_%d", group.IDGroup))
		inlineKeyboard = append(inlineKeyboard, tgbotapi.NewInlineKeyboardRow(button))
	}

	// Отправляем сообщение с кнопками
	msg := tgbotapi.NewMessage(chatID, "Выберите группу для мероприятия:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(inlineKeyboard...)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}

	// Сохраняем шаг выбора группы
	userSteps[chatID] = "selecting_event_group"
	log.Printf("Переход к состоянию: %s", userSteps[chatID])
}

func handleEventCreation(bot *tgbotapi.BotAPI, chatID int64, text string) {
	event := tempEvent[chatID]

	switch userSteps[chatID] {
	case "selecting_event_group":
		log.Println("Ожидался выбор группы через инлайн-кнопку.")
		msg := tgbotapi.NewMessage(chatID, "Пожалуйста, выберите группу, нажав на кнопку.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}

	case "creating_event_category":
		validCategories := []string{"Личное", "Семья", "Работа"}
		isValid := false
		for _, category := range validCategories {
			if text == category {
				isValid = true
				break
			}
		}
		if !isValid {
			msg := tgbotapi.NewMessage(chatID, "Некорректная категория. Пожалуйста, выберите из: Личное, Семья, Работа.")
			if _, err := bot.Send(msg); err != nil {
				log.Printf("Ошибка отправки сообщения: %v", err)
			}
			return
		}
		event.Category = text
		tempEvent[chatID] = event
		userSteps[chatID] = "creating_event_name"
		log.Printf("Переход к состоянию: %s", userSteps[chatID])
		msg := tgbotapi.NewMessage(chatID, "Введите название мероприятия:")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}

	case "creating_event_name":
		event.NameEvent = text
		tempEvent[chatID] = event
		userSteps[chatID] = "creating_event_time"
		log.Printf("Переход к состоянию: %s", userSteps[chatID])
		msg := tgbotapi.NewMessage(chatID, "Введите дату и время начала в формате дд.мм.гггг чч:мм или нажмите 'Весь день':")
		msg.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
			Keyboard: [][]tgbotapi.KeyboardButton{
				{tgbotapi.NewKeyboardButton("Весь день"), tgbotapi.NewKeyboardButton("Главное меню")},
			},
			ResizeKeyboard: true,
		}
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}

	case "creating_event_time":
		if strings.HasPrefix(text, "Весь день") {
			userSteps[chatID] = "creating_event_all_day_date"
			log.Printf("Переход к состоянию: %s", userSteps[chatID])
			msg := tgbotapi.NewMessage(chatID, "Введите дату для мероприятия в формате дд.мм.гггг:")
			if _, err := bot.Send(msg); err != nil {
				log.Printf("Ошибка отправки сообщения: %v", err)
			}
			return
		}
		layout := "02.01.2006 15:04"
		startTime, err := time.Parse(layout, text)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Неверный формат. Пожалуйста, введите дату и время в формате дд.мм.гггг чч:мм.")
			if _, err := bot.Send(msg); err != nil {
				log.Printf("Ошибка отправки сообщения: %v", err)
			}
			return
		}
		event.DatetimeStart = startTime
		tempEvent[chatID] = event
		userSteps[chatID] = "creating_event_duration"
		log.Printf("Переход к состоянию: %s", userSteps[chatID])
		msg := tgbotapi.NewMessage(chatID, "Введите продолжительность мероприятия (например, 1d2h) или нажмите 'Пропустить':")
		msg.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
			Keyboard: [][]tgbotapi.KeyboardButton{
				{tgbotapi.NewKeyboardButton("Пропустить"), tgbotapi.NewKeyboardButton("Главное меню")},
			},
			ResizeKeyboard: true,
		}
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}

	case "creating_event_all_day_date":
		layout := "02.01.2006"
		allDayDate, err := time.Parse(layout, text)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Неверный формат. Пожалуйста, введите дату в формате дд.мм.гггг.")
			if _, err := bot.Send(msg); err != nil {
				log.Printf("Ошибка отправки сообщения: %v", err)
			}
			return
		}
		event.DatetimeStart = allDayDate
		event.IsAllDay = true
		tempEvent[chatID] = event
		userSteps[chatID] = "creating_event_duration"
		log.Printf("Переход к состоянию: %s", userSteps[chatID])
		msg := tgbotapi.NewMessage(chatID, "Введите продолжительность мероприятия (например, 1d2h) или нажмите 'Пропустить':")
		msg.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
			Keyboard: [][]tgbotapi.KeyboardButton{
				{tgbotapi.NewKeyboardButton("Пропустить"), tgbotapi.NewKeyboardButton("Главное меню")},
			},
			ResizeKeyboard: true,
		}
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}

	case "creating_event_duration":
		if text != "Пропустить" {
			duration, err := parseDuration(text)
			if err != nil {
				msg := tgbotapi.NewMessage(chatID, "Неверный формат продолжительности. Используйте формат 1d2h3m.")
				if _, err := bot.Send(msg); err != nil {
					log.Printf("Ошибка отправки сообщения: %v", err)
				}
				return
			}
			event.Duration = duration
		}

		event.Status = "Запланировано"

		if err := db.DB.Create(&event).Error; err != nil {
			log.Println("Ошибка сохранения события:", err)
			msg := tgbotapi.NewMessage(chatID, "Ошибка при сохранении события.")
			if _, err := bot.Send(msg); err != nil {
				log.Printf("Ошибка отправки сообщения: %v", err)
			}
			return
		}

		delete(tempEvent, chatID)
		delete(userSteps, chatID)

		log.Println("Мероприятие успешно создано.")
		msg := tgbotapi.NewMessage(chatID, "Мероприятие успешно создано!")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
		sendMainMenu(bot, chatID)
	}
}

// ---- Функционал создания группы ----
func startCreateGroup(bot *tgbotapi.BotAPI, chatID int64) {
	userSteps[chatID] = "creating_group_name"
	tempGroup[chatID] = gorm_models.Group{}

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
	if text == "Главное меню" {
		delete(tempGroup, chatID)
		delete(userSteps, chatID)
		sendMainMenu(bot, chatID)
		return
	}

	group := tempGroup[chatID]

	switch userSteps[chatID] {
	case "creating_group_name":
		group.GroupName = text
		tempGroup[chatID] = group
		userSteps[chatID] = "adding_group_members"

		msg := tgbotapi.NewMessage(chatID, "Введите теги участников группы (через запятую):")
		bot.Send(msg)

	case "adding_group_members":
		newGroup := gorm_models.Group{GroupName: group.GroupName}
		if err := db.DB.Create(&newGroup).Error; err != nil {
			log.Println("Ошибка сохранения группы:", err)
			bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при создании группы."))
			return
		}

		// Добавляем администратора в таблицу `Membership`
		adminMembership := gorm_models.Membership{
			IDGroup: newGroup.IDGroup,
			IDUser:  chatID,
			IDAdmin: chatID,
		}
		if err := db.DB.Create(&adminMembership).Error; err != nil {
			log.Println("Ошибка добавления администратора:", err)
			bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при добавлении администратора группы."))
			return
		}

		// Добавление участников
		participants := strings.Split(text, ",")
		for _, participant := range participants {
			participant = strings.TrimSpace(participant)
			participantID, ok := authorizedUsers[participant]
			if !ok || participantID == chatID { // Игнорируем неавторизованных и администратора
				continue
			}

			membership := gorm_models.Membership{
				IDGroup: newGroup.IDGroup,
				IDUser:  participantID,
				IDAdmin: chatID, // ID администратора группы
			}
			if err := db.DB.Create(&membership).Error; err != nil {
				log.Printf("Ошибка добавления участника %s: %v", participant, err)
			}
		}

		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Группа '%s' успешно создана!", group.GroupName))
		bot.Send(msg)

		delete(tempGroup, chatID)
		delete(userSteps, chatID)
		sendMainMenu(bot, chatID)
	}
}

// func startGroupEventCreation(bot *tgbotapi.BotAPI, chatID int64) {
// 	var memberships []gorm_models.Membership
//
// 	// Извлекаем группы, в которых состоит пользователь
// 	if err := db.DB.Where("id_user = ?", chatID).Find(&memberships).Error; err != nil {
// 		log.Println("Ошибка получения групп пользователя:", err)
// 		msg := tgbotapi.NewMessage(chatID, "Ошибка при получении ваших групп.")
// 		bot.Send(msg)
// 		return
// 	}
//
// 	// Собираем ID групп
// 	groupIDs := make([]int64, 0)
// 	for _, membership := range memberships {
// 		groupIDs = append(groupIDs, membership.IDGroup)
// 	}
//
// 	// Если пользователь не состоит ни в одной группе
// 	if len(groupIDs) == 0 {
// 		msg := tgbotapi.NewMessage(chatID, "У вас пока нет групп. Создайте группу перед добавлением мероприятия.")
// 		bot.Send(msg)
// 		return
// 	}
//
// 	// Извлекаем информацию о группах по их ID
// 	var groups []gorm_models.Group
// 	if err := db.DB.Where("id_group IN ?", groupIDs).Find(&groups).Error; err != nil {
// 		log.Println("Ошибка получения информации о группах:", err)
// 		msg := tgbotapi.NewMessage(chatID, "Ошибка при получении данных о группах.")
// 		bot.Send(msg)
// 		return
// 	}
//
// 	// Создаем клавиатуру с кнопками для выбора группы
// 	var inlineKeyboard [][]tgbotapi.InlineKeyboardButton
// 	for _, group := range groups {
// 		button := tgbotapi.NewInlineKeyboardButtonData(group.GroupName, fmt.Sprintf("group_%d", group.IDGroup))
// 		inlineKeyboard = append(inlineKeyboard, tgbotapi.NewInlineKeyboardRow(button))
// 	}
//
// 	// Отправляем сообщение с кнопками, если группы найдены
// 	if len(inlineKeyboard) > 0 {
// 		msg := tgbotapi.NewMessage(chatID, "Выберите группу для создания мероприятия:")
// 		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(inlineKeyboard...)
// 		bot.Send(msg)
// 	} else {
// 		// На случай, если группы неожиданно не найдены
// 		msg := tgbotapi.NewMessage(chatID, "У вас пока нет групп. Создайте группу перед добавлением мероприятия.")
// 		bot.Send(msg)
// 	}
// }

func handleCallbackQuery(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID

	// Если callback data начинается с "group_", значит это выбор группы
	if strings.HasPrefix(callback.Data, "group_") {
		groupID, err := strconv.Atoi(strings.TrimPrefix(callback.Data, "group_"))
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Некорректный выбор группы.")
			bot.Send(msg)
			return
		}
		var group gorm_models.Group
		if err := db.DB.First(&group, groupID).Error; err != nil {
			msg := tgbotapi.NewMessage(chatID, "Некорректный выбор группы.")
			bot.Send(msg)
			return
		}

		// Сохраняем выбранную группу и переходим к созданию мероприятия
		userSteps[chatID] = "creating_event_for_group"
		groupID64 := int64(groupID)
		tempEvent[chatID] = gorm_models.Event{
			IDGroup:  groupID64,
			Category: "Группа " + group.GroupName,
		}

		userSteps[chatID] = "creating_event_category" // Устанавливаем шаг выбора категории
		if err := db.DB.First(&group, groupID).Error; err != nil {
			msg := tgbotapi.NewMessage(chatID, "Группа не найдена.")
			bot.Send(msg)
			return
		}
		tempEvent[chatID] = gorm_models.Event{
			IDGroup:  groupID64,
			Category: "Группа " + group.GroupName,
		}

		msg := tgbotapi.NewMessage(chatID, "Выберите категорию мероприятия:")
		msg.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
			Keyboard: [][]tgbotapi.KeyboardButton{
				{tgbotapi.NewKeyboardButton("Личное"), tgbotapi.NewKeyboardButton("Семья"), tgbotapi.NewKeyboardButton("Работа")},
				{tgbotapi.NewKeyboardButton("Главное меню")},
			},
			ResizeKeyboard: true,
		}
		bot.Send(msg)

		// Уведомляем Telegram о завершении обработки callback
		bot.Request(tgbotapi.NewCallback(callback.ID, "Группа выбрана!"))
		return
	}

	// Если callback не распознан
	bot.Request(tgbotapi.NewCallback(callback.ID, "Неизвестное действие"))
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
