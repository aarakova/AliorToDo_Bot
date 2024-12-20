package gorm_models

type Membership struct {
	IDGroup int64 `gorm:"foreignKey:IDGroup;references:IDGroup;column:id_group;not null"`
	IDUser  int64 `gorm:"foreignKey:IDUser;references:IDUser;column:id_user;not null"`
	IDAdmin int64 `gorm:"column:id_admin;not null"`
}
