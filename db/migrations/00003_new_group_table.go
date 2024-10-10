package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upNewGroupTable, downNewGroupTable)
}

func upNewGroupTable(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	exec, err := tx.ExecContext(ctx, `
		CREATE TABLE todo_group(
    		id_group SERIAL PRIMARY KEY,
    		group_name VARCHAR(255) NOT NULL
		);
	`)
	if err != nil {
		return err
	} else if rowsAffected, _ := exec.RowsAffected(); rowsAffected == 0 {
		return fmt.Errorf("opperation CREATE TABLE todo_group doesn't change anything")
	}
	return nil
}

func downNewGroupTable(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	exec, err := tx.ExecContext(ctx, `
		DROP TABLE todo_group;
	`)
	if err != nil {
		return err
	} else if rowsAffected, _ := exec.RowsAffected(); rowsAffected == 0 {
		return fmt.Errorf("opperation 'DROP TABLE todo_group' doesn't change anything")
	}
	return nil
}
