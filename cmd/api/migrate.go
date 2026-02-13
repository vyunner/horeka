package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func runMigrateCmd(args []string) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required in .env")
	}

	sourceURL := "file://./migrations"

	m, err := migrate.New(sourceURL, dsn)
	if err != nil {
		log.Fatalf("migrate init: %v", err)
	}

	if len(args) == 0 {
		printMigrateHelp()
		return
	}

	cmd := args[0]

	switch cmd {
	case "up":
		err = ignoreNoChange(m.Up())

	case "down":
		steps := 1
		if len(args) >= 2 {
			n, convErr := strconv.Atoi(args[1])
			if convErr != nil || n <= 0 {
				log.Fatal("down steps must be a positive integer")
			}
			steps = n
		}
		err = ignoreNoChange(m.Steps(-steps))

	case "reset":
		err = ignoreNoChange(m.Down())

	case "steps":
		if len(args) < 2 {
			log.Fatal("steps requires a number: steps 1 or steps -1")
		}
		n, convErr := strconv.Atoi(args[1])
		if convErr != nil || n == 0 {
			log.Fatal("steps must be a non-zero integer")
		}
		err = ignoreNoChange(m.Steps(n))

	case "to":
		if len(args) < 2 {
			log.Fatal("to requires a version number: to 2")
		}
		v, convErr := strconv.Atoi(args[1])
		if convErr != nil || v < 0 {
			log.Fatal("version must be a non-negative integer")
		}
		err = ignoreNoChange(m.Migrate(uint(v)))

	case "version":
		v, dirty, verr := m.Version()
		if verr != nil {
			if errors.Is(verr, migrate.ErrNilVersion) {
				fmt.Println("version: none (no migrations applied)")
				return
			}
			log.Fatalf("version: %v", verr)
		}
		fmt.Printf("version: %d dirty: %v\n", v, dirty)
		return

	case "force":
		if len(args) < 2 {
			log.Fatal("force requires a version number: force 2")
		}
		v, convErr := strconv.Atoi(args[1])
		if convErr != nil || v < 0 {
			log.Fatal("force version must be a non-negative integer")
		}
		if err := m.Force(v); err != nil {
			log.Fatalf("migrate force: %v", err)
		}
		fmt.Printf("migrate force %d: OK\n", v)
		return

	default:
		printMigrateHelp()
		return
	}

	if err != nil {
		log.Fatalf("migrate %s: %v", cmd, err)
	}
	fmt.Printf("migrate %s: OK\n", cmd)
}

func ignoreNoChange(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}
	return err
}

func printMigrateHelp() {
	fmt.Println(`Usage:
  go run .\cmd\api migrate up
  go run .\cmd\api migrate down [steps]    (default 1)
  go run .\cmd\api migrate reset          (DOWN ALL)
  go run .\cmd\api migrate steps <N>      (N can be negative, N != 0)
  go run .\cmd\api migrate to <version>   (migrate to exact version)
  go run .\cmd\api migrate version
  go run .\cmd\api migrate force <version>
`)
}
