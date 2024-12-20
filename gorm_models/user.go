package gorm_models

type User struct {
	IDUser   int64  `gorm:"primaryKey;type:text"`
	UserName string `gorm:"type:text;not null"`
	IDChat   int64  `gorm:"autoIncrement"`
}
