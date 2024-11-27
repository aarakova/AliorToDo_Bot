package tg_provider

import (
	`context`
)

type DbProvider interface {
	DeleteEvent(ctx context.Context, id int32) error
}
