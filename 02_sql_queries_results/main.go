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

	// Создание таблицы
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		age INTEGER
	)`)
	if err != nil {
		log.Fatal(err)
	}

	// Вставка данных
	_, err = db.Exec("INSERT INTO users (name, age) VALUES (?, ?)", "Alexey", 45)
	if err != nil {
		log.Fatal(err)
	}

	// Выборка данных
	rows, err := db.Query("SELECT name, age FROM users WHERE age > ?", 25)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var age int
		if err := rows.Scan(&name, &age); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Имя: %s, Возраст: %d\n", name, age)
	}
}
