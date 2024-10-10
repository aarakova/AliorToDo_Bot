package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upNewEventTable, downNewEventTable)
}

func upNewEventTable(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	exec, err := tx.ExecContext(ctx, `
		CREATE TABLE todo_event(
    		id_event SERIAL PRIMARY KEY,
    		id_group SERIAL, 
    		category VARCHAR(255) NOT NULL,
    		name_event TEXT NOT NULL,
    		datatime_start TIMESTAMP NOT NULL,
    		duration INTERVAL NOT NULL,
    		link_to_video TEXT NOT NULL,
    		status VARCHAR(255) NOT NULL,
    		FOREIGN KEY (id_group) REFERENCES todo_group(id_group)		                    
		);
	`)
	if err != nil {
		return err
	} else if rowsAffected, _ := exec.RowsAffected(); rowsAffected == 0 {
		return fmt.Errorf("opperation CREATE TABLE todo_event doesn't change anything")
	}
	return nil
}

func downNewEventTable(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	exec, err := tx.ExecContext(ctx, `
		DROP TABLE todo_event;
	`)
	if err != nil {
		return err
	} else if rowsAffected, _ := exec.RowsAffected(); rowsAffected == 0 {
		return fmt.Errorf("opperation 'DROP TABLE todo_event' doesn't change anything")
	}
	return nil
}
