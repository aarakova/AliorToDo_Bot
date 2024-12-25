package gorm_models

type Membership struct {
	IDGroup int64 `gorm:"foreignKey:IDGroup;references:IDGroup;column:id_group;not null"`
	IDUser  int64 `gorm:"foreignKey:IDUser;references:IDUser;column:id_user;not null"`
	IsAdmin bool  `gorm:"column:is_admin;not null"`
}
