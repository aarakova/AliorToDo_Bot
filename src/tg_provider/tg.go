package tg_provider

import (
	"context"
	"time"
)

type EventProvider interface {
	CreateEvent(ctx context.Context,
		idGroup int32, category, nameEvent string, timeStart time.Time,
		duration time.Duration, linkToVideo, status string) error
	DeleteEvent(ctx context.Context, id int32) error
	UpdateEvent(ctx context.Context,
		nameEvent string, timeStart time.Time, duration time.Duration,
		linkToVideo, status string, idEvent int32) error
}

type GroupProvider interface {
	CreateGroup(ctx context.Context, groupName string) error
	DeleteGroup(ctx context.Context, id int32) error
	UpdateGroup(ctx context.Context, nameGroup string, idGroup int32) error
}

type MembershipProvider interface {
	CreateMembership(ctx context.Context, idGroup int32, idUser, idAdmin string) error
	DeleteMembership(ctx context.Context, idGroup int32, idUser string) error
}

type UserProvider interface {
	CreateUser(ctx context.Context, idUser, userName string, idChat int32) error
	DeleteUser(ctx context.Context, idUser int32) error
	UpdateUser(ctx context.Context, userName, idUser string) error
}

type DbProvider interface {
	EventProvider
	GroupProvider
	MembershipProvider
	UserProvider
}
