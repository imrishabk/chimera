package main

import (
	"database/sql"
	"os"
	"strings"

	"charm.land/log/v2"
	"github.com/imrishabk/chimera/services/worker/internal/util"
	"github.com/imrishabk/chimera/services/worker/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
)

func init() {
	godotenv.Load()
}

func main() {
	cmd, args := "up", os.Args
	if len(args) > 2 {
		cmd = strings.ToLower(args[1])
	}
	dsn := util.ConstructDSN()
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal(err)
	}

	goose.SetBaseFS(migrations.Files)
	switch cmd {
	case "up":
		if err = goose.Up(conn, "."); err != nil {
			log.Fatal(err)
		}
	case "down":
		if err = goose.Down(conn, "."); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("invalid command (use up to run the migration or down to remove migrations")
	}
}
