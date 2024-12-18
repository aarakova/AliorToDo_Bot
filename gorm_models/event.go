package gorm_models

import (
	"time"
)

type Event struct {
	IDEvent       int       `gorm:"primaryKey;autoIncrement"`
	IDGroup       int       `gorm:"not null"`
	Category      string    `gorm:"type:text;not null"`
	NameEvent     string    `gorm:"type:text;not null"`
	DatetimeStart time.Time `gorm:"not null"`
	Duration      time.Duration
	LinkToVideo   string `gorm:"type:text"`
	Status        string `gorm:"type:text;not null"`

	Group Group `gorm:"foreignKey:IDGroup;constraint:OnDelete:CASCADE;"`
}
