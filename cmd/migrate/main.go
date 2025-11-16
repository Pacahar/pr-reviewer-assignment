package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"github.com/pacahar/pr-reviewer-assignment/internal/config"
)

func main() {
	config := config.MustLoad()

	db, err := sql.Open("postgres", config.Database.DSN())

	if err != nil {
		panic(err)
	}
	defer db.Close()

	data, err := os.ReadFile("../../migrations/init.sql")
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(string(data))
	if err != nil {
		panic(err)
	}

	fmt.Println("Migration applied!")
}
