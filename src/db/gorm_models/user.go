package gorm_models

type User struct {
	IDUser   int64  `gorm:"column:id_user;primaryKey"`
	UserName string `gorm:"column:user_name;type:text;not null"`
	IDChat   int64  `gorm:"column:id_chat;autoIncrement"`
}
