package gorm_models

type Group struct {
	IDGroup   int    `gorm:"primaryKey;autoIncrement"`
	GroupName string `gorm:"type:text;not null"`
}
