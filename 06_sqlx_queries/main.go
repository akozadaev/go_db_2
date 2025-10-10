package main

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
	Age  int    `db:"age"`
}

func main() {
	db, err := sqlx.Connect("sqlite3", "./sqlx_test.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		age INTEGER
	)`)
	if err != nil {
		log.Fatal(err)
	}

	// Вставка
	user := User{Name: "Alexey", Age: 45}
	_, err = db.NamedExec("INSERT INTO users (name, age) VALUES (:name, :age)", user)
	if err != nil {
		log.Fatal(err)
	}

	// Чтение
	var users []User
	err = db.Select(&users, "SELECT * FROM users")
	if err != nil {
		log.Fatal(err)
	}

	for _, u := range users {
		fmt.Printf("ID: %d, Имя: %s, Возраст: %d\n", u.ID, u.Name, u.Age)
	}
}
