package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"aliorToDoBot/src/db/gorm_models"
)

var (
	errNoGroup    = fmt.Errorf("группа не найдена")
	errNoCategory = fmt.Errorf("категория не найдена")
	errNoUser     = fmt.Errorf("пользователь не найден")
	errInternal   = fmt.Errorf("системная ошибка")
)

type DatabaseProvider interface {
	providerUser
	providerGroup
	providerEvent
}

type providerGroup interface {
	GetGroup(IDGroup int64) (string, error)
	CreateGroup(GroupName string, usernames []string) error
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

type GormProvider struct {
	*gorm.DB
	IDUser    int64
	UserName  string
	GroupName string
}

// GetGroup возвращает название группы по её ID.
// Если группа не найдена, возвращается ошибка errNoGroup.
func (g *GormProvider) GetGroup(IDGroup int64) (string, error) {
	var groups []gorm_models.Group
	if err := g.Where("id_group IN ?", IDGroup).Find(&groups).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errNoGroup
		}
		return "", errInternal
	}
	return g.GroupName, nil
}

// CreateGroup создает новую группу с указанным именем и добавляет в неё пользователей.
// Если chatID пользователя совпадает, то он назначается администратором.
func (g *GormProvider) CreateGroup(ctx context.Context, chatID int64, groupName string, usernames []string) error {
	var users []gorm_models.User
	tx := g.WithContext(ctx).Where("user_name IN ?", usernames).Find(&users)
	newGroup := &gorm_models.Group{
		GroupName: groupName,
	}
	tx.WithContext(ctx).Create(newGroup)
	for _, v := range users {
		isAdmin := v.IDChat == chatID
		tx.WithContext(ctx).Create(&gorm_models.Membership{
			IDGroup: newGroup.IDGroup,
			IDUser:  v.IDUser,
			IDAdmin: isAdmin,
		})
	}
	return tx.Error
}

// DeleteGroup удаляет группу с указанным названием.
// Удаляются также все записи участников этой группы.
func (g *GormProvider) DeleteGroup(ctx context.Context, chatID int64, groupName string) error {
	var group gorm_models.Group

	if err := g.WithContext(ctx).Where("group_name = ?", groupName).First(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errNoGroup
		}
		return errInternal
	}

	isAdmin, err := g.isAdmin(ctx, chatID, group.IDGroup)
	if err != nil {
		return err
	}
	if !isAdmin {
		return fmt.Errorf("только администратор может удалить группу")
	}

	if err = g.WithContext(ctx).Where("id_group = ?", group.IDGroup).Delete(&gorm_models.Membership{}).Error; err != nil {
		return errInternal
	}

	return g.WithContext(ctx).Delete(&group).Error
}

func (g *GormProvider) GetUser(chatID int64) (int64, string, error) {
	var user gorm_models.User

	if err := g.Where("id_chat = ?", chatID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, "", errNoUser
		}
		return 0, "", errInternal
	}
	return g.IDUser, g.UserName, nil
}

// CreateUser создает нового пользователя с указанными chatID и именем.
// Если пользователь с данным chatID уже существует, возвращается ошибка.
func (g *GormProvider) CreateUser(ctx context.Context, chatID int64, userName string) error {
	if err := g.WithContext(ctx).Where("id_chat = ?", chatID).First(&gorm_models.User{}).Error; err == nil {
		return fmt.Errorf("пользователь с chatID %d уже существует", chatID)
	}

	if err := g.WithContext(ctx).Create(&gorm_models.User{
		IDChat:   chatID,
		UserName: userName,
	}).Error; err != nil {
		return errInternal
	}

	return nil
}

// GetEvents возвращает список событий для пользователя с указанным chatID.
// Возвращается строковое представление событий, если они найдены.
func (g *GormProvider) GetEvents(ctx context.Context, chatID int64) (string, error) {
	var user gorm_models.User
	if err := g.WithContext(ctx).Where("id_chat = ?", chatID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errNoUser
		}
		return "", errInternal
	}

	var events []gorm_models.Event
	if err := g.WithContext(ctx).Where("id_group IN (?)",
		g.WithContext(ctx).Model(&gorm_models.Membership{}).
			Select("id_group").
			Where("id_user = ?", user.IDUser),
	).Find(&events).Error; err != nil {
		return "", errInternal
	}

	if len(events) == 0 {
		return "Нет запланированных событий", nil
	}
	var result string
	for _, event := range events {
		result += fmt.Sprintf("Событие: %s, Категория: %s, Начало: %s\n",
			event.NameEvent, event.Category, event.DatetimeStart.Format("02.01.2006 15:04"))
	}

	return result, nil
}

// CreateEvent создает новое событие для указанной группы.
// Проверяется наличие категории и принадлежность пользователя к группе.
func (g *GormProvider) CreateEvent(ctx context.Context, chatID int64, groupName, nameEvent, category string,
	isAllDay bool, datetimeStart time.Time, duration time.Duration) error {
	var (
		validCategories = []string{"Личное", "Семья", "Работа"}
		isValid         bool
	)

	for _, validCategory := range validCategories {
		if category == validCategory {
			isValid = true
			break
		}
	}
	if !isValid {
		return errNoCategory
	}

	var group gorm_models.Group
	if err := g.WithContext(ctx).Where("group_name = ?", groupName).First(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errNoGroup
		}
		return errInternal
	}

	isAdmin, err := g.isAdmin(ctx, chatID, group.IDGroup)
	if err != nil {
		return err
	}
	if !isAdmin {
		return fmt.Errorf("только администратор может создавать события")
	}

	// Создаем событие
	newEvent := &gorm_models.Event{
		NameEvent:     nameEvent,
		IDGroup:       group.IDGroup,
		DatetimeStart: datetimeStart,
		Category:      category,
		Duration:      duration,
		IsAllDay:      isAllDay,
		Status:        "Запланировано",
	}

	return g.WithContext(ctx).Create(newEvent).Error
}

// DeleteEvent удаляет событие с указанным именем.
// Проверяется, что пользователь является администратором группы.
func (g *GormProvider) DeleteEvent(ctx context.Context, chatID int64, nameEvent string) error {
	var event gorm_models.Event

	if err := g.WithContext(ctx).Where("name_event = ?", nameEvent).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("событие с именем '%s' не найдено", nameEvent)
		}
		return errInternal
	}

	isAdmin, err := g.isAdmin(ctx, chatID, event.IDGroup)
	if err != nil {
		return err
	}
	if !isAdmin {
		return fmt.Errorf("только администратор может удалить событие")
	}

	return g.WithContext(ctx).Delete(&event).Error
}

// isAdmin проверяет, является ли пользователь администратором указанной группы.
func (g *GormProvider) isAdmin(ctx context.Context, chatID int64, groupID int64) (bool, error) {
	var membership gorm_models.Membership
	if err := g.WithContext(ctx).
		Where("id_group = ? AND id_user = ? AND id_admin = true", groupID, chatID).
		First(&membership).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, errInternal
	}
	return true, nil
}
