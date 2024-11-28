package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CoreDBProvider struct {
	db *pgxpool.Pool
}

// CreateEvent - the function that create event in database
func (c *CoreDBProvider) CreateEvent(ctx context.Context,
	idGroup int32, category, nameEvent string, timeStart time.Time, duration time.Duration, linkToVideo, status string) error {
	createQuery := `
	INSERT INTO todo_event (id_group, category, name_event, datatime_start,
	                        duration, link_to_video, status) 
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	exec, err := c.db.Exec(ctx,
		createQuery, idGroup, category, nameEvent, timeStart, duration, linkToVideo, status)
	if err != nil {
		return err
	}
	if exec.RowsAffected() == 0 {
		return fmt.Errorf("запись не была добавлена")
	}
	return nil
}
func (c *CoreDBProvider) DeleteEvent(ctx context.Context, id int32) error {
	deleteQuery := `DELETE FROM todo_event WHERE id_event = $1`
	exec, err := c.db.Exec(ctx, deleteQuery, id)
	if err != nil {
		return err
	}
	if exec.RowsAffected() == 0 {
		return fmt.Errorf("запись не была удалена")
	}
	return nil
}

func (c *CoreDBProvider) UpdateEvent(ctx context.Context,
	nameEvent string, timeStart time.Time, duration time.Duration, linkToVideo, status string, idEvent int32) error {
	createQuery := `
	UPDATE todo_event
	SET name_event = $1, 
	    datatime_start = $2, 
	    duration = $3, 
	    link_to_video = $4, 
	    status = $5
	WHERE id_event = $6
	`
	exec, err := c.db.Exec(ctx,
		createQuery, nameEvent, timeStart, duration, linkToVideo, status, idEvent)
	if err != nil {
		return err
	}
	if exec.RowsAffected() == 0 {
		return fmt.Errorf("запись не была изменена")
	}
	return nil
}

func (c *CoreDBProvider) CreateGroup(ctx context.Context, groupName string) error {
	createQuery := `
	INSERT INTO todo_group (group_name) 
	VALUES ($1)
	`
	exec, err := c.db.Exec(ctx,
		createQuery, groupName)
	if err != nil {
		return err
	}
	if exec.RowsAffected() == 0 {
		return fmt.Errorf("группа не была создана")
	}
	return nil
}

func (c *CoreDBProvider) DeleteGroup(ctx context.Context, id int32) error {
	deleteQuery := `DELETE FROM todo_group WHERE id_group = $1`
	exec, err := c.db.Exec(ctx, deleteQuery, id)
	if err != nil {
		return err
	}
	if exec.RowsAffected() == 0 {
		return fmt.Errorf("группа не была удалена")
	}
	return nil
}

func (c *CoreDBProvider) UpdateGroup(ctx context.Context,
	nameGroup string, idGroup int32) error {
	createQuery := `
	UPDATE todo_group
	SET group_name = $1
	WHERE id_group = $2
	`
	exec, err := c.db.Exec(ctx,
		createQuery, nameGroup, idGroup)
	if err != nil {
		return err
	}
	if exec.RowsAffected() == 0 {
		return fmt.Errorf("группа не была изменена")
	}
	return nil
}

func (c *CoreDBProvider) CreateMembership(ctx context.Context,
	idGroup int32, idUser, idAdmin string) error {
	createQuery := `
	INSERT INTO todo_membership (id_group, id_user, id_admin) 
	VALUES ($1, $2, $3)
	`
	exec, err := c.db.Exec(ctx,
		createQuery, idGroup, idUser, idAdmin)
	if err != nil {
		return err
	}
	if exec.RowsAffected() == 0 {
		return fmt.Errorf("пользователь не был добавлен")
	}
	return nil
}

func (c *CoreDBProvider) DeleteMembership(ctx context.Context,
	idGroup int32, idUser string) error {
	deleteQuery := `DELETE FROM todo_membership WHERE id_group = $1 AND id_user = $2`
	exec, err := c.db.Exec(ctx, deleteQuery, idGroup, idUser)
	if err != nil {
		return err
	}
	if exec.RowsAffected() == 0 {
		return fmt.Errorf("участник не был удален")
	}
	return nil
}

func (c *CoreDBProvider) CreateUser(ctx context.Context,
	idUser, userName string, idChat int32) error {
	createQuery := `
	INSERT INTO todo_user (id_user, user_name, id_chat) 
	VALUES ($1, $2, $3)
	`
	exec, err := c.db.Exec(ctx,
		createQuery, idUser, userName, idChat)
	if err != nil {
		return err
	}
	if exec.RowsAffected() == 0 {
		return fmt.Errorf("пользователь не был зарегистрирован")
	}
	return nil
}

func (c *CoreDBProvider) DeleteUser(ctx context.Context,
	idUser int32) error {
	deleteQuery := `DELETE FROM todo_user WHERE id_user = $1`
	exec, err := c.db.Exec(ctx, deleteQuery, idUser)
	if err != nil {
		return err
	}
	if exec.RowsAffected() == 0 {
		return fmt.Errorf("пользователь не был удален")
	}
	return nil
}

func (c *CoreDBProvider) UpdateUser(ctx context.Context,
	userName, idUser string) error {
	createQuery := `
	UPDATE todo_user
	SET user_name = $1
	WHERE id_user = $2
	`
	exec, err := c.db.Exec(ctx,
		createQuery, userName, idUser)
	if err != nil {
		return err
	}
	if exec.RowsAffected() == 0 {
		return fmt.Errorf("имя пользователя не было изменено")
	}
	return nil
}
