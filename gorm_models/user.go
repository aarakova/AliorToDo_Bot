package gorm_models

type User struct {
	IDUser   string `gorm:"primaryKey;type:text"`
	UserName string `gorm:"type:text;not null"`
	IDChat   int    `gorm:"type:serial"`
}
