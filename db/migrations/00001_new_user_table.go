package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upNewUserTable, downNewUserTable)
}

func upNewUserTable(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	exec, err := tx.ExecContext(ctx, `
		CREATE TABLE todo_user(
    		id_user VARCHAR(255) NOT NULL PRIMARY KEY,
    		user_name VARCHAR(255) NOT NULL,
    		id_chat SERIAL
		);
	`)
	if err != nil {
		return err
	} else if rowsAffected, _ := exec.RowsAffected(); rowsAffected == 0 {
		return fmt.Errorf("opperation CREATE TABLE todo_user doesn't change anything")
	}
	return nil
}

func downNewUserTable(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	exec, err := tx.ExecContext(ctx, `
		DROP TABLE todo_user;
	`)
	if err != nil {
		return err
	} else if rowsAffected, _ := exec.RowsAffected(); rowsAffected == 0 {
		return fmt.Errorf("opperation 'DROP TABLE todo_user' doesn't change anything")
	}
	return nil
}
