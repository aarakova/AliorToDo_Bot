package main

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"

	"aliorToDoBot/gorm_models"
	"aliorToDoBot/src/db"
)

var (
	userSteps = make(map[int64]string)
	tempEvent = make(map[int64]gorm_models.Event) // Временное хранилище для событий на этапе создания
	tempGroup = make(map[int64]gorm_models.Group) // Временное хранилище для групп на этапе создания
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
				handleDefault(bot, chatID, update.Message.Text, update.Message.Chat.UserName)
			}
		}
	}
}

// ---- Общие функции ----
func handleDefault(bot *tgbotapi.BotAPI, chatID int64, text, username string) {
	fmt.Println("Получено сообщение:", text)

	if text == "Главное меню" {
		delete(userSteps, chatID)
		sendMainMenu(bot, chatID)
		return
	}

	switch text {
	case "/start":
		checkAndAddNewUser(username, chatID)
		ensurePersonalGroup(bot, chatID)
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
		UpdateEventStatuses(db.DB)
		viewMyEvents(bot, chatID)
	case "Удалить мероприятие":
		deleteEvent(bot, chatID)
	case "Мои группы":
		viewMyGroups(bot, chatID)
	case "Выйти из группы":
		leaveGroup(bot, chatID)
	default:
		msg := tgbotapi.NewMessage(chatID, "Неизвестная команда. Используйте /start.")
		bot.Send(msg)
	}
}

func checkAndAddNewUser(username string, chatID int64) {
	err := db.DB.Where("id_user = ?", chatID).First(
		&gorm_models.User{IDChat: chatID}).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Ошибка проверки авторизации пользователя %d: %v", chatID, err)
		return
	}

	// Если группа "Личное" уже существует, выходим
	if err == nil {
		log.Printf("Пользователь уже существует %d", chatID)
		return
	}

	err = db.DB.Create(&gorm_models.User{IDChat: chatID, UserName: username}).Error
	if err != nil {
		log.Println("Не смог создать новго пользователя")
	}
}

func ensurePersonalGroup(bot *tgbotapi.BotAPI, chatID int64) {
	// Извлекаем IDUser из таблицы users по chatID
	var user gorm_models.User
	err := db.DB.Where("id_chat = ?", chatID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Пользователь с chatID %d не найден", chatID)
			msg := tgbotapi.NewMessage(chatID, "Ваш пользовательский профиль не найден. Обратитесь в поддержку.")
			bot.Send(msg)
			return
		}
		log.Printf("Ошибка извлечения пользователя с chatID %d: %v", chatID, err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		bot.Send(msg)
		return
	}

	// Проверяем, есть ли уже группа "Личное" для данного пользователя
	var group gorm_models.Group
	err = db.DB.Where("group_name = ? AND id_group IN (SELECT id_group FROM memberships WHERE id_user = ?)", "Личное", user.IDUser).First(&group).Error

	// Если произошла ошибка, но она не связана с отсутствием записи, логируем и выходим
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Ошибка проверки группы 'Личное' для пользователя %d: %v", user.IDUser, err)
		return
	}

	// Если группа "Личное" уже существует, выходим
	if err == nil {
		log.Printf("Группа 'Личное' уже существует для пользователя %d", user.IDUser)
		return
	}

	// Создаем группу "Личное"
	newGroup := gorm_models.Group{
		GroupName: "Личное",
	}

	if err = db.DB.Create(&newGroup).Error; err != nil {
		log.Printf("Ошибка создания группы 'Личное' для пользователя %d: %v", user.IDUser, err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка при создании группы 'Личное'. Попробуйте позже.")
		bot.Send(msg)
		return
	}

	// Добавляем запись о членстве (Membership) для администратора
	membership := gorm_models.Membership{
		IDGroup: newGroup.IDGroup, // ID группы
		IDUser:  user.IDUser,      // ID пользователя из таблицы users
		IDAdmin: user.IDUser,      // Администратор — текущий пользователь
	}

	if err = db.DB.Create(&membership).Error; err != nil {
		log.Printf("Ошибка добавления пользователя %d в группу 'Личное': %v", user.IDUser, err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка при создании группы 'Личное'. Попробуйте позже.")
		bot.Send(msg)
		return
	}

	log.Printf("Группа 'Личное' успешно создана для пользователя %d", user.IDUser)
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
		bot.Send(msg)
		return
	}

	if len(memberships) == 0 {
		msg := tgbotapi.NewMessage(chatID, "У вас пока нет мероприятий.")
		bot.Send(msg)
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
		bot.Send(msg)
		return
	}

	if len(events) == 0 {
		msg := tgbotapi.NewMessage(chatID, "У вас пока нет мероприятий.")
		bot.Send(msg)
		return
	}

	// Извлекаем данные о группах для отображения
	var groups []gorm_models.Group
	if err := db.DB.Where("id_group IN ?", groupIDs).Find(&groups).Error; err != nil {
		log.Println("Ошибка получения данных групп:", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка при обработке данных ваших групп.")
		bot.Send(msg)
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
		message.WriteString(formatEvent(event, groupName) + "\n\n")
	}

	// Отправляем сообщение с клавиатурой
	msg := tgbotapi.NewMessage(chatID, message.String())
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
		Keyboard: [][]tgbotapi.KeyboardButton{
			{tgbotapi.NewKeyboardButton("Удалить мероприятие")},
			{tgbotapi.NewKeyboardButton("Главное меню")},
		},
		ResizeKeyboard: true,
	}
	bot.Send(msg)
}

func formatEvent(event gorm_models.Event, groupName string) string {
	// Форматируем продолжительность без секунд
	formattedDuration := formatDuration(event.Duration)

	if event.IsAllDay {
		return fmt.Sprintf("📅 *%s*\nГруппа: %s\nКатегория: %s\nДата: %s\nСтатус: %s",
			event.NameEvent, groupName, event.Category, event.DatetimeStart.Format("02.01.2006"), event.Status)
	}
	return fmt.Sprintf("📅 *%s*\nГруппа: %s\nКатегория: %s\nДата и время: %s\nПродолжительность: %s\nСтатус: %s",
		event.NameEvent, groupName, event.Category, event.DatetimeStart.Format("02.01.2006 15:04"), formattedDuration, event.Status)
}

// Функция форматирования продолжительности без секунд
func formatDuration(d time.Duration) string {
	days := d / (24 * time.Hour) // Вычисляем дни
	d -= days * 24 * time.Hour   // Убираем дни из общей продолжительности
	hours := d / time.Hour       // Вычисляем часы
	d -= hours * time.Hour       // Убираем часы
	minutes := d / time.Minute   // Вычисляем минуты

	// Формируем строку
	var result string
	if days > 0 {
		result += fmt.Sprintf("%dd", days)
	}
	if hours > 0 {
		if result != "" {
			result += " "
		}
		result += fmt.Sprintf("%dh", hours)
	}
	if minutes > 0 {
		if result != "" {
			result += " "
		}
		result += fmt.Sprintf("%dm", minutes)
	}

	return result
}

func deleteEvent(bot *tgbotapi.BotAPI, chatID int64) {
	// Получаем список мероприятий пользователя
	var events []gorm_models.Event
	err := db.DB.Where("id_group IN (SELECT id_group FROM memberships WHERE id_user = ?)", chatID).Find(&events).Error
	if err != nil {
		log.Printf("Ошибка получения мероприятий: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка при получении списка мероприятий.")
		bot.Send(msg)
		return
	}

	if len(events) == 0 {
		msg := tgbotapi.NewMessage(chatID, "У вас пока нет мероприятий для удаления.")
		bot.Send(msg)
		return
	}

	// Создаем инлайн-кнопки для мероприятий
	var inlineKeyboard [][]tgbotapi.InlineKeyboardButton
	for _, event := range events {
		button := tgbotapi.NewInlineKeyboardButtonData(event.NameEvent, fmt.Sprintf("delete_event_%d", event.IDEvent))
		inlineKeyboard = append(inlineKeyboard, tgbotapi.NewInlineKeyboardRow(button))
	}

	msg := tgbotapi.NewMessage(chatID, "Выберите мероприятие для удаления:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(inlineKeyboard...)
	bot.Send(msg)
}

// ---- Функционал просмотра групп ----
func viewMyGroups(bot *tgbotapi.BotAPI, chatID int64) {
	// Извлекаем IDUser из таблицы users по chatID
	var user gorm_models.User
	if err := db.DB.Where("id_chat = ?", chatID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Пользователь с chatID %d не найден", chatID)
			bot.Send(tgbotapi.NewMessage(chatID, "Ваш пользовательский профиль не найден. Обратитесь в поддержку."))
			return
		}
		log.Printf("Ошибка извлечения пользователя с chatID %d: %v", chatID, err)
		bot.Send(tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже."))
		return
	}

	// Получение записей Membership для пользователя
	var memberships []gorm_models.Membership
	if err := db.DB.Where("id_user = ?", user.IDUser).Find(&memberships).Error; err != nil {
		log.Println("Ошибка получения membership записей:", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка при получении ваших групп.")
		bot.Send(msg)
		return
	}

	// Проверка, есть ли группы у пользователя
	if len(memberships) == 0 {
		msg := tgbotapi.NewMessage(chatID, "У вас пока нет групп.")
		msg.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
			Keyboard: [][]tgbotapi.KeyboardButton{
				{tgbotapi.NewKeyboardButton("Главное меню")},
			},
			ResizeKeyboard: true,
		}
		bot.Send(msg)
		return
	}

	// Получение ID групп из Membership
	groupIDs := make([]int64, 0)
	for _, membership := range memberships {
		groupIDs = append(groupIDs, membership.IDGroup)
	}

	// Получение информации о группах
	var groups []gorm_models.Group
	if err := db.DB.Where("id_group IN ?", groupIDs).Find(&groups).Error; err != nil {
		log.Println("Ошибка получения групп:", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка при получении ваших групп.")
		bot.Send(msg)
		return
	}

	// Формирование сообщения с информацией о группах
	var message strings.Builder
	message.WriteString("Ваши группы:\n\n")
	for _, group := range groups {
		// Получение всех участников группы
		var groupMemberships []gorm_models.Membership
		if err := db.DB.Where("id_group = ?", group.IDGroup).Find(&groupMemberships).Error; err != nil {
			log.Printf("Ошибка получения участников для группы %d: %v", group.IDGroup, err)
			continue
		}

		// Формирование списка участников и администратора
		members := make([]string, 0)
		admin := ""

		for _, membership := range groupMemberships {
			var groupUser gorm_models.User
			if err := db.DB.Where("id_user = ?", membership.IDUser).First(&groupUser).Error; err == nil {
				if membership.IDAdmin == membership.IDUser {
					admin = "@" + groupUser.UserName
				} else {
					members = append(members, "@"+groupUser.UserName)
				}
			} else {
				log.Printf("Ошибка получения пользователя: IDUser=%d, Ошибка: %v", membership.IDUser, err)
			}
		}

		// Если администратор не найден, добавить сообщение в лог
		if admin == "" {
			log.Printf("Администратор для группы '%s' (ID: %d) не найден!", group.GroupName, group.IDGroup)
			admin = "Не указан"
		}

		// Добавление информации о группе в сообщение
		message.WriteString(fmt.Sprintf(
			"Группа: %s\nАдминистратор: %s\nУчастники: %s\n\n",
			group.GroupName,
			admin,
			strings.Join(members, ", "),
		))
	}

	// Отправка сообщения с информацией о группах
	msg := tgbotapi.NewMessage(chatID, message.String())
	msg.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
		Keyboard: [][]tgbotapi.KeyboardButton{
			{tgbotapi.NewKeyboardButton("Выйти из группы")},
			{tgbotapi.NewKeyboardButton("Главное меню")},
		},
		ResizeKeyboard: true,
	}
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
	if text == "Главное меню" {
		delete(tempGroup, chatID)
		delete(userSteps, chatID)
		sendMainMenu(bot, chatID)
		return
	}

	switch userSteps[chatID] {
	case "selecting_event_group":
		if text == "Главное меню" {
			userSteps[chatID] = ""
			return
		}
		log.Println("Ожидался выбор группы через инлайн-кнопку.")
		msg := tgbotapi.NewMessage(chatID, "Пожалуйста, выберите группу, нажав на кнопку.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}

	case "creating_event_category":
		if text == "Главное меню" {
			userSteps[chatID] = ""
			return
		}
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
		if text == "Главное меню" {
			userSteps[chatID] = ""
			return
		}
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
		if text == "Главное меню" {
			userSteps[chatID] = ""
			return
		}
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
		event.DatetimeStart = startTime.UTC()
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
		if text == "Главное меню" {
			userSteps[chatID] = ""
			return
		}
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
		if _, err = bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}

	case "creating_event_duration":
		if text == "Главное меню" {
			userSteps[chatID] = ""
			return
		}
		if text != "Пропустить" { // Если пользователь не пропускает ввод
			duration, err := parseDuration(text) // Парсим продолжительность
			if err != nil {                      // Если формат некорректен
				log.Printf("Ошибка парсинга продолжительности: %v", err)
				msg := tgbotapi.NewMessage(chatID, "Неверный формат продолжительности. Используйте формат 1d2h3m, где:\n- d: дни\n- h: часы\n- m: минуты. Пример: 1d2h или 2h30m.")
				if _, err = bot.Send(msg); err != nil {
					log.Printf("Ошибка отправки сообщения: %v", err)
				}
				return // Прерываем выполнение, чтобы пользователь ввёл данные заново
			}
			event.Duration = duration // Сохраняем продолжительность, если формат корректен
		} else {
			event.Duration = 0 // Если пользователь пропустил, устанавливаем продолжительность как 0
		}

		event.Status = "Запланировано"

		// Сохраняем событие в базу данных
		if err := db.DB.Create(&event).Error; err != nil {
			log.Println("Ошибка сохранения события:", err)
			msg := tgbotapi.NewMessage(chatID, "Ошибка при сохранении события.")
			if _, err := bot.Send(msg); err != nil {
				log.Printf("Ошибка отправки сообщения: %v", err)
			}
			return
		}

		delete(tempEvent, chatID) // Удаляем временные данные
		delete(userSteps, chatID) // Сбрасываем шаги

		log.Println("Мероприятие успешно создано.")
		msg := tgbotapi.NewMessage(chatID, "Мероприятие успешно создано!")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}

		sendMainMenu(bot, chatID) // Возвращаем пользователя в главное меню
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
		// Извлекаем IDUser из таблицы пользователей по chatID
		var creator gorm_models.User
		if err := db.DB.Where("id_chat = ?", chatID).First(&creator).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("Создатель группы с chatID %d не найден", chatID)
				bot.Send(tgbotapi.NewMessage(chatID, "Ваш пользовательский профиль не найден. Обратитесь в поддержку."))
				return
			}
			log.Printf("Ошибка извлечения пользователя с chatID %d: %v", chatID, err)
			bot.Send(tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже."))
			return
		}

		// Создаем группу
		newGroup := gorm_models.Group{GroupName: group.GroupName}
		if err := db.DB.Create(&newGroup).Error; err != nil {
			log.Println("Ошибка сохранения группы:", err)
			bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при создании группы."))
			return
		}

		// Добавляем администратора в таблицу `Membership`
		adminMembership := gorm_models.Membership{
			IDGroup: newGroup.IDGroup,
			IDUser:  creator.IDUser, // IDUser создателя
			IDAdmin: creator.IDUser, // Устанавливаем администратора
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
			participant = strings.TrimPrefix(participant, "@")

			var user gorm_models.User
			if err := db.DB.Where("user_name = ?", participant).First(&user).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) || user.IDUser == creator.IDUser { // Игнорируем несуществующих пользователей и администратора
					continue
				}
				log.Printf("Ошибка проверки пользователя %s: %v", participant, err)
				continue
			}

			membership := gorm_models.Membership{
				IDGroup: newGroup.IDGroup,
				IDUser:  user.IDUser,
				IDAdmin: creator.IDUser, // Устанавливаем создателя группы как администратора
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
func leaveGroup(bot *tgbotapi.BotAPI, chatID int64) {
	// Извлекаем IDUser из таблицы users по chatID
	var user gorm_models.User
	if err := db.DB.Where("id_chat = ?", chatID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Пользователь с chatID %d не найден", chatID)
			if _, err := bot.Send(tgbotapi.NewMessage(chatID, "Ваш пользовательский профиль не найден. Обратитесь в поддержку.")); err != nil {
				log.Printf("Ошибка отправки сообщения: %v", err)
			}
			return
		}
		log.Printf("Ошибка извлечения пользователя с chatID %d: %v", chatID, err)
		if _, err := bot.Send(tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже.")); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
		return
	}

	// Получаем группы, в которых пользователь состоит, исключая группу "Личное"
	var memberships []gorm_models.Membership
	err := db.DB.Where("id_user = ?", user.IDUser).Find(&memberships).Error
	if err != nil {
		log.Printf("Ошибка получения membership записей: %v", err)
		if _, err := bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при получении списка групп.")); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
		return
	}

	var groups []gorm_models.Group
	for _, membership := range memberships {
		var group gorm_models.Group
		if err := db.DB.First(&group, membership.IDGroup).Error; err == nil && group.GroupName != "Личное" {
			groups = append(groups, group)
		}
	}

	// Если у пользователя нет доступных групп
	if len(groups) == 0 {
		msg := tgbotapi.NewMessage(chatID, "У вас нет групп, из которых можно выйти.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
		return
	}

	// Создаем инлайн-кнопки для выбора группы
	var inlineKeyboard [][]tgbotapi.InlineKeyboardButton
	for _, group := range groups {
		button := tgbotapi.NewInlineKeyboardButtonData(group.GroupName, fmt.Sprintf("leave_group_%d", group.IDGroup))
		inlineKeyboard = append(inlineKeyboard, tgbotapi.NewInlineKeyboardRow(button))
	}

	// Отправляем сообщение с инлайн-кнопками
	msg := tgbotapi.NewMessage(chatID, "Выберите группу для выхода:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(inlineKeyboard...)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

func handleCallbackQuery(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	data := callback.Data

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

	// Обработка удаления мероприятия
	if strings.HasPrefix(data, "delete_event_") {
		eventID, err := strconv.Atoi(strings.TrimPrefix(data, "delete_event_"))
		if err != nil {
			log.Printf("Ошибка преобразования ID мероприятия: %v", err)
			bot.Request(tgbotapi.NewCallback(callback.ID, "Некорректный ID мероприятия."))
			return
		}

		var event gorm_models.Event
		err = db.DB.First(&event, eventID).Error
		if err != nil {
			log.Printf("Ошибка получения мероприятия: %v", err)
			bot.Request(tgbotapi.NewCallback(callback.ID, "Мероприятие не найдено."))
			return
		}

		// Запрос подтверждения удаления
		confirmationText := fmt.Sprintf("Удалить мероприятие '%s'?", event.NameEvent)
		confirmKeyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Да", fmt.Sprintf("confirm_delete_%d", eventID)),
				tgbotapi.NewInlineKeyboardButtonData("Нет", "cancel_delete"),
			),
		)

		msg := tgbotapi.NewMessage(chatID, confirmationText)
		msg.ReplyMarkup = confirmKeyboard
		bot.Send(msg)
		return
	}

	// Обработка подтверждения удаления
	if strings.HasPrefix(data, "confirm_delete_") {
		eventID, err := strconv.Atoi(strings.TrimPrefix(data, "confirm_delete_"))
		if err != nil {
			log.Printf("Ошибка преобразования ID мероприятия: %v", err)
			bot.Request(tgbotapi.NewCallback(callback.ID, "Некорректный ID мероприятия."))
			return
		}

		err = db.DB.Delete(&gorm_models.Event{}, eventID).Error
		if err != nil {
			log.Printf("Ошибка удаления мероприятия: %v", err)
			bot.Request(tgbotapi.NewCallback(callback.ID, "Не удалось удалить мероприятие."))
			return
		}

		bot.Request(tgbotapi.NewCallback(callback.ID, "Мероприятие успешно удалено."))
		msg := tgbotapi.NewMessage(chatID, "Мероприятие успешно удалено.")
		bot.Send(msg)
		viewMyEvents(bot, chatID)
		return
	}

	// Отмена удаления
	if data == "cancel_delete" {
		bot.Request(tgbotapi.NewCallback(callback.ID, "Удаление отменено."))
		msg := tgbotapi.NewMessage(chatID, "Удаление отменено.")
		bot.Send(msg)
		viewMyEvents(bot, chatID)
		return
	}

	// выход из группы
	if strings.HasPrefix(data, "leave_group_") {
		groupID, err := strconv.Atoi(strings.TrimPrefix(data, "leave_group_"))
		if err != nil {
			log.Printf("Ошибка обработки ID группы: %v", err)
			bot.Request(tgbotapi.NewCallback(callback.ID, "Некорректный выбор группы."))
			return
		}
		// поиск группы по ID
		var group gorm_models.Group
		if err := db.DB.First(&group, groupID).Error; err != nil {
			log.Printf("Ошибка получения группы с ID %d: %v", groupID, err)
			bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка при получении данных группы."))
			return
		}

		// Запрос подтверждения выхода
		confirmText := fmt.Sprintf("Покинуть группу \"%s\"?", group.GroupName)
		confirmKeyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Да", fmt.Sprintf("confirm_leave_%d", groupID)),
				tgbotapi.NewInlineKeyboardButtonData("Нет", "cancel_leave"),
			),
		)

		msg := tgbotapi.NewMessage(chatID, confirmText)
		msg.ReplyMarkup = confirmKeyboard
		bot.Send(msg)
		return
	}

	if strings.HasPrefix(data, "confirm_leave_") {
		groupID, err := strconv.Atoi(strings.TrimPrefix(data, "confirm_leave_"))
		if err != nil {
			log.Printf("Ошибка обработки ID группы: %v", err)
			bot.Request(tgbotapi.NewCallback(callback.ID, "Некорректный выбор группы."))
			return
		}

		// Извлекаем IDUser из таблицы users по chatID
		var user gorm_models.User
		if err := db.DB.Where("id_chat = ?", chatID).First(&user).Error; err != nil {
			log.Printf("Ошибка извлечения пользователя с chatID %d: %v", chatID, err)
			bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка при выходе из группы."))
			return
		}

		var membership gorm_models.Membership
		if err := db.DB.Where("id_group = ? AND id_user = ?", groupID, user.IDUser).First(&membership).Error; err != nil {
			log.Printf("Ошибка получения membership записи: %v", err)
			bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка при выходе из группы."))
			return
		}

		// Удаляем пользователя из группы
		if err := db.DB.Where("id_group = ? AND id_user = ?", groupID, user.IDUser).Delete(&gorm_models.Membership{}).Error; err != nil {
			log.Printf("Ошибка выхода из группы: %v", err)
			bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка при выходе из группы."))
			return
		}

		// Проверяем количество оставшихся участников
		var remainingMembers int64
		if err := db.DB.Model(&gorm_models.Membership{}).Where("id_group = ?", groupID).Count(&remainingMembers).Error; err != nil {
			log.Printf("Ошибка проверки участников группы: %v", err)
			bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка при проверке участников группы."))
			return
		}

		if remainingMembers == 1 {
			// Удаляем группу, если участников больше нет
			if err := db.DB.Delete(&gorm_models.Group{}, groupID).Error; err != nil {
				log.Printf("Ошибка удаления группы с ID %d: %v", groupID, err)
				bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка при удалении группы."))
				return
			}

			bot.Request(tgbotapi.NewCallback(callback.ID, "Группа успешно удалена."))
			msg := tgbotapi.NewMessage(chatID, "Группа удалена, так как в ней не осталось участников.")
			bot.Send(msg)
			viewMyGroups(bot, chatID)
			return
		}

		bot.Request(tgbotapi.NewCallback(callback.ID, "Вы успешно покинули группу."))
		msg := tgbotapi.NewMessage(chatID, "Вы успешно покинули группу.")
		bot.Send(msg)
		viewMyGroups(bot, chatID)
	}

	if data == "cancel_leave" {
		bot.Request(tgbotapi.NewCallback(callback.ID, "Выход из группы отменен."))
		msg := tgbotapi.NewMessage(chatID, "Выход из группы отменен.")
		bot.Send(msg)
		viewMyGroups(bot, chatID)
		return
	}

	// Если callback не распознан
	bot.Request(tgbotapi.NewCallback(callback.ID, "Неизвестное действие"))
}

func parseDuration(input string) (time.Duration, error) {
	// Регулярное выражение для проверки формата
	re := regexp.MustCompile(`^(\d+d)?(\d+h)?(\d+m)?$`)
	if !re.MatchString(input) {
		return 0, errors.New("некорректный формат продолжительности")
	}
	matches := re.FindStringSubmatch(input)
	if matches == nil {
		return 0, errors.New("не могу найти дату")
	}
	var duration time.Duration
	if matches[1] != "" { // Если дни найдены
		days, err := strconv.Atoi(strings.TrimSuffix(matches[1], "d"))
		if err != nil {
			return 0, errors.New("некорректный формат дней")
		}
		duration += time.Duration(days) * 24 * time.Hour
	}
	// Парсим часы
	if matches[2] != "" { // Если часы найдены
		hours, err := strconv.Atoi(strings.TrimSuffix(matches[2], "h"))
		if err != nil {
			return 0, errors.New("некорректный формат часов")
		}
		duration += time.Duration(hours) * time.Hour
	}

	// Парсим минуты
	if matches[3] != "" { // Если минуты найдены
		minutes, err := strconv.Atoi(strings.TrimSuffix(matches[3], "m"))
		if err != nil {
			return 0, errors.New("некорректный формат минут")
		}
		duration += time.Duration(minutes) * time.Minute
	}

	return duration, nil
}

// Обновление статуса мероприятия в зависимости от его времени и продолжительности
func UpdateEventStatuses(db *gorm.DB) {
	// Получаем все мероприятия из базы
	var events []gorm_models.Event
	err := db.Find(&events).Error
	if err != nil {
		log.Printf("Ошибка получения мероприятий: %v", err)
		return
	}

	// Получаем текущее время
	localTime := time.Now()

	// Извлекаем компоненты локального времени
	year, month, day := localTime.Date()
	hour, min, sec := localTime.Clock()

	// Создаем новое время с часовым поясом UTC, но используя компоненты локального времени
	currentTime := time.Date(year, month, day, hour, min, sec, localTime.Nanosecond(), time.UTC)

	for _, event := range events {
		previousStatus := event.Status

		startTime := event.DatetimeStart.UTC()
		var endTime time.Time
		if event.Duration > 0 {
			endTime = startTime.Add(event.Duration)
		} else {
			endTime = startTime // Если продолжительность равна 0, конец совпадает с началом
		}

		log.Printf("Проверяем мероприятие ID: %d, StartTime: %v, EndTime: %v, CurrentTime: %v", event.IDEvent, startTime, endTime, currentTime)

		// Логика определения статуса
		if currentTime.Before(startTime) {
			event.Status = "Запланировано"
		} else if currentTime.After(endTime) {
			event.Status = "Завершено"
		} else if currentTime.After(startTime) && currentTime.Before(endTime) {
			event.Status = "В процессе"
		}

		log.Printf("Статус мероприятия ID: %d изменился с '%s' на '%s'", event.IDEvent, previousStatus, event.Status)

		// Обновляем статус в базе, если он изменился
		if previousStatus != event.Status {
			err := db.Model(&gorm_models.Event{}).
				Where("id_event = ?", event.IDEvent).
				Update("status", event.Status).Error
			if err != nil {
				log.Printf("Ошибка обновления статуса мероприятия ID %d: %v", event.IDEvent, err)
			} else {
				log.Printf("Статус мероприятия ID %d обновлен на '%s'", event.IDEvent, event.Status)
			}
		}
	}
}
