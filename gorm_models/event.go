package gorm_models

import (
	"time"
)

type Event struct {
	IDEvent       int64         `gorm:"primaryKey;autoIncrement"`
	NameEvent     string        `gorm:"not null"`
	IDGroup       int64         `gorm:"foreignKey:IDGroup;references:IDGroup;not null"`
	DatetimeStart time.Time     `gorm:"column:datetime_start"`
	Category      string        `gorm:"not null;check:category IN ('Личное','Семья','Работа')"`
	Duration      time.Duration `gorm:"not null"`
	IsAllDay      bool          `gorm:"not null"`
	Status        string        `gorm:"not null; check:status IN ('Запланировано', 'В работе', 'Завершено')"`
}
