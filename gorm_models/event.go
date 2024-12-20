package gorm_models

import (
	"time"
)

type Event struct {
	IDEvent       int64         `gorm:"primaryKey;autoIncrement"`
	NameEvent     string        `gorm:"not null"`
	IDGroup       int64         `gorm:"foreignKey:IDGroup;references:IDGroup;not null"`
	DatetimeStart time.Time     `gorm:"type:timestamp without time zone;column:datetime_start"`
	Category      string        `gorm:"not null;check:category IN ('Личное','Семья','Работа')"`
	Duration      time.Duration `gorm:"column:duration"`
	IsAllDay      bool          `gorm:"not null"`
	Status        string        `gorm:"not null; check:status IN ('Запланировано', 'В работе', 'Завершено')"`
}
