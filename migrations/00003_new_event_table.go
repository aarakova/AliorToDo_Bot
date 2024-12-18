package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upNewEventTable, downNewEventTable)
}

func upNewEventTable(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.ExecContext(ctx, `
		CREATE TABLE todo_event(
    		id_event SERIAL PRIMARY KEY,
    		id_group SERIAL, 
    		category event_category NOT NULL,
    		name_event text NOT NULL,
    		datatime_start TIMESTAMP NOT NULL,
    		duration INTERVAL NOT NULL,
    		link_to_video text NOT NULL,
    		status event_status NOT NULL,
    		FOREIGN KEY (id_group) REFERENCES todo_group(id_group)		                    
		);
	`)
	if err != nil {
		return err
	}
	return nil
}

func downNewEventTable(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.ExecContext(ctx, `
		DROP TABLE todo_event;
	`)
	if err != nil {
		return err
	}
	return nil
}
