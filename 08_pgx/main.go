package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "ibs"
	password = "ibs"
	dbname   = "ibs"
)

func main() {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", user, password, host, port, dbname)

	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v\n", err)
	}
	defer conn.Close(context.Background())

	fmt.Println("Успешное подключение к PostgreSQL!")

	createTableSQL := `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL
		);`

	_, err = conn.Exec(context.Background(), createTableSQL)
	if err != nil {
		log.Fatalf("Ошибка при создании таблицы: %v\n", err)
	}
	fmt.Println("Таблица 'users' создана или уже существует.")

	insertSQL := "INSERT INTO users (name, email) VALUES ($1, $2) ON CONFLICT DO NOTHING;"
	_, err = conn.Exec(context.Background(), insertSQL, "Иван Иванов", "ivan@example.com")
	if err != nil {
		log.Fatalf("Ошибка при вставке данных: %v\n", err)
	}
	_, err = conn.Exec(context.Background(), insertSQL, "Мария Машина", "maria@example.com")
	if err != nil {
		log.Fatalf("Ошибка при вставке данных: %v\n", err)
	}

	fmt.Println("Данные успешно вставлены.")

	rows, err := conn.Query(context.Background(), "SELECT id, name, email FROM users;")
	if err != nil {
		log.Fatalf("Ошибка при выполнении запроса: %v\n", err)
	}
	defer rows.Close()

	fmt.Println("\nСодержимое таблицы 'users':")
	for rows.Next() {
		var id int
		var name, email string
		err = rows.Scan(&id, &name, &email)
		if err != nil {
			log.Fatalf("Ошибка при чтении строки: %v\n", err)
		}
		fmt.Printf("ID: %d, Имя: %s, Email: %s\n", id, name, email)
	}

	if err = rows.Err(); err != nil {
		log.Fatalf("Ошибка при итерации по строкам: %v\n", err)
	}

	fmt.Println("\n Пример завершён успешно!")
}
