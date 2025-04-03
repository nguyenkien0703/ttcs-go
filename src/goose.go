package main

import (
	"application/src/config"
	lib_db "application/src/lib/db"
	"flag"
	"github.com/pressly/goose"
	"log"
	"os"
)

var (
	flags = flag.NewFlagSet("goose", flag.ExitOnError)
	dir   = flags.String("dir", ".", "directory with migration files")
)

func gooseMain() {
	flags.Usage = usage
	flags.Parse(os.Args[1:])
	args := flags.Args()
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		flags.Usage()
		return
	}

	switch args[0] {
	case "create":
		if err := goose.Run("create", nil, *dir, args[1:]...); err != nil {
			log.Fatalf("goose run: %v", err)
		}
		return
	case "fix":
		if err := goose.Run("fix", nil, *dir); err != nil {
			log.Fatalf("goose run: %v", err)
		}
		return
	}
	dbClient, err := lib_db.Connect(config.DbDefault, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer lib_db.Disconnect(dbClient)
	db, err := dbClient.GetDB().DB()
	if err != nil {
		log.Fatal(err)
	}
	connectionSetting := lib_db.GetConnectionSetting(config.DbDefault)
	driver := "mysql"

	switch driver {
	case "postgres", "mysql":
		if err := goose.SetDialect(driver); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("%q driver not supported\n", driver)
	}
	dbstring := connectionSetting.Address()
	command := args[len(args)-1]
	switch dbstring {
	case "":
		log.Fatalf("-dbstring=%q not supported\n", dbstring)
	default:
	}
	arguments := []string{}
	if len(args) > 3 {
		arguments = append(arguments, args[3:]...)
	}
	if err := goose.Run(command, db, *dir, arguments...); err != nil {
		log.Fatalf("goose run: %v", err)
	}
}

func usage() {
	log.Print(usagePrefix)
	flags.PrintDefaults()
	log.Print(usageCommands)
}

var (
	usagePrefix = `Usage: goose [OPTIONS] DRIVER DBSTRING COMMAND
Drivers:
    postgres
    mysql
Examples:
    goose mysql ./foo.db status
    goose mysql ./foo.db create init sql
    goose mysql ./foo.db create add_some_column sql
    goose mysql ./foo.db create fetch_user_data go
    goose mysql ./foo.db up
    goose postgres "user=postgres dbname=postgres sslmode=disable" status
    goose mysql "user:password@/dbname?parseTime=true" status
    goose redshift "postgres://user:password@qwerty.us-east-1.redshift.amazonaws.com:5439/db" status
Options:
`

	usageCommands = `
Commands:
    up                   Migrate the DB to the most recent version available
    up-to VERSION        Migrate the DB to a specific VERSION
    down                 Roll back the version by 1
    down-to VERSION      Roll back to a specific VERSION
    redo                 Re-run the latest migration
    status               Dump the migration status for the current DB
    version              Print the current version of the database
    create NAME [sql|go] Creates new migration file with the current timestamp
		fix                  Apply sequential ordering to migrations
`
)
