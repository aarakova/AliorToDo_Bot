package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upNewGroupTable, downNewGroupTable)
}

func upNewGroupTable(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.ExecContext(ctx, `
		CREATE TABLE todo_group(
    		id_group SERIAL PRIMARY KEY,
    		group_name VARCHAR(255) NOT NULL
		);
	`)
	if err != nil {
		return err
	}
	return nil
}

func downNewGroupTable(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.ExecContext(ctx, `
		DROP TABLE todo_group;
	`)
	if err != nil {
		return err
	}
	return nil
}
