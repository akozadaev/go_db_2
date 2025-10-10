package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "ibs"
	password = "ibs"
	dbname   = "testdb"
)

func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Ошибка при открытии подключения: %v\n", err)
	}
	defer db.Close()

	// Проверка подключения
	if err = db.Ping(); err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v\n", err)
	}
	fmt.Println("Успешное подключение к PostgreSQL!")

	// Создание таблицы
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL
	)`)
	if err != nil {
		log.Fatalf("Ошибка при создании таблицы: %v\n", err)
	}
	fmt.Println("Таблица 'users' создана или уже существует.")

	// Вставка данных
	_, err = db.Exec("INSERT INTO users (name, email) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		"Alexey Kozadaev", "akozadaev@inbox.ru")
	if err != nil {
		log.Fatalf("Ошибка при вставке данных: %v\n", err)
	}
	fmt.Println("Данные вставлены.")

	// Выборка данных
	rows, err := db.Query("SELECT id, name, email FROM users")
	if err != nil {
		log.Fatalf("Ошибка при выполнении SELECT: %v\n", err)
	}
	defer rows.Close()

	fmt.Println("\nСодержимое таблицы 'users':")
	for rows.Next() {
		var id int
		var name, email string
		if err := rows.Scan(&id, &name, &email); err != nil {
			log.Fatalf("Ошибка при чтении строки: %v\n", err)
		}
		fmt.Printf("ID: %d | Имя: %s | Email: %s\n", id, name, email)
	}

	if err = rows.Err(); err != nil {
		log.Fatalf("Ошибка при итерации по результатам: %v\n", err)
	}
}
