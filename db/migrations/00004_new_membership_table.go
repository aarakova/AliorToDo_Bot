package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upNewMembershipTable, downNewMembershipTable)
}

func upNewMembershipTable(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	exec, err := tx.ExecContext(ctx, `
		CREATE TABLE todo_membership(
    		id_group SERIAL,
    		id_user VARCHAR(255) NOT NULL,
    		id_admin VARCHAR(255) NOT NULL,
    		FOREIGN KEY (id_group) REFERENCES todo_group(id_group),
    		FOREIGN KEY (id_user) REFERENCES todo_user(id_user)	
		);
	`)
	if err != nil {
		return err
	} else if rowsAffected, _ := exec.RowsAffected(); rowsAffected == 0 {
		return fmt.Errorf("opperation CREATE TABLE todo_membership doesn't change anything")
	}
	return nil
}

func downNewMembershipTable(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	exec, err := tx.ExecContext(ctx, `
		DROP TABLE todo_membership;
	`)
	if err != nil {
		return err
	} else if rowsAffected, _ := exec.RowsAffected(); rowsAffected == 0 {
		return fmt.Errorf("opperation 'DROP TABLE todo_membership' doesn't change anything")
	}
	return nil
}
