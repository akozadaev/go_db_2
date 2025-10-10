package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "./test.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback() // откат при панике или ошибке

	_, err = tx.Exec("INSERT INTO users (name, age) VALUES (?, ?)", "Bob", 28)
	if err != nil {
		log.Fatal(err)
	}

	_, err = tx.Exec("INSERT INTO users (name, age) VALUES (?, ?)", "Charlie", 35)
	if err != nil {
		log.Fatal(err)
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Транзакция успешно завершена")
}
