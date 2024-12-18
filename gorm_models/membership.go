package gorm_models

type Membership struct {
	IDGroup int    `gorm:"not null"`
	IDUser  string `gorm:"not null"`
	IDAdmin string `gorm:"not null"`

	Group Group `gorm:"foreignKey:IDGroup;constraint:OnDelete:CASCADE;"`
	User  User  `gorm:"foreignKey:IDUser;constraint:OnDelete:CASCADE;"`
}
