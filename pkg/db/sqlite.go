package db

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func OpenSQLite(dsn string) (*sql.DB, error) {
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	database.SetMaxOpenConns(1)
	if err := database.Ping(); err != nil {
		_ = database.Close()
		return nil, err
	}
	return database, nil
}

func Migrate(database *sql.DB, statements ...string) error {
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
