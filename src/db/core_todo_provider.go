package db

import (
	`context`
	`fmt`
	`time`

	`github.com/jackc/pgx/v5/pgxpool`
)

type CoreDBProvider struct {
	db *pgxpool.Pool
}

// CreateEvent - the function that create event in database
func (c *CoreDBProvider) CreateEvent(ctx context.Context,
	idGroup int32, category, nameEvent string, timeStart time.Time) error {
	createQuery := `
	INSERT INTO todo_event (id_group, category, name_event, datatime_start,
	                        duration, link_to_video, status) 
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	// TODO: дописать аргументы везде
	exec, err := c.db.Exec(ctx,
		createQuery, idGroup, category, nameEvent, timeStart)
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

// TODO: операция UPDATE
