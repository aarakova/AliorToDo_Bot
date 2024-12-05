package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upNewMembershipTable, downNewMembershipTable)
}

func upNewMembershipTable(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.ExecContext(ctx, `
		CREATE TABLE todo_membership(
    		id_group SERIAL,
    		id_user text NOT NULL,
    		id_admin text NOT NULL,
    		FOREIGN KEY (id_group) REFERENCES todo_group(id_group),
    		FOREIGN KEY (id_user) REFERENCES todo_user(id_user)	
		);
	`)
	if err != nil {
		return err
	}
	return nil
}

func downNewMembershipTable(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.ExecContext(ctx, `
		DROP TABLE todo_membership;
	`)
	if err != nil {
		return err
	}
	return nil
}
