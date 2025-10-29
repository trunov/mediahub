package migrate

import (
	"database/sql"
	"embed"
	"io/fs"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations
var Migrations embed.FS

func Migrate(dsn string, path fs.FS) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return err
	}

	goose.SetBaseFS(path)
	return goose.Up(db, "migrations")
}
