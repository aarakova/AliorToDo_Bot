package gorm_models

type Group struct {
	IDGroup   int64  `gorm:"primaryKey;autoIncrement"`
	GroupName string `gorm:"not null"`
}
