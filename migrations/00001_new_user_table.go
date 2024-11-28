package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upNewUserTable, downNewUserTable)
}

func upNewUserTable(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.ExecContext(ctx, `
		CREATE TABLE todo_user(
    		id_user text NOT NULL PRIMARY KEY,
    		user_name text NOT NULL,
    		id_chat SERIAL
		);
	`)
	if err != nil {
		return err
	}
	return nil
}

func downNewUserTable(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.ExecContext(ctx, `
		DROP TABLE todo_user;
	`)
	if err != nil {
		return err
	}
	return nil
}
