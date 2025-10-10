package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/001_init_tables.sql
var migrationSQL string

const connStr = "postgres://ibs:ibs@localhost:5432/ibs_migration?sslmode=disable"

// Account представляет аккаунт
type Account struct {
	Username string
	Email    string
}

// Валидация аккаунта на стороне Go
func (a *Account) Validate() error {
	if len(a.Username) < 3 {
		return fmt.Errorf("username must be at least 3 characters")
	}
	if len(a.Username) > 50 {
		return fmt.Errorf("username too long")
	}
	if !isValidEmail(a.Email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func main() {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("❌ Не удалось создать пул: %v", err)
	}
	defer pool.Close()

	// Применяем миграцию
	err = applyMigration(ctx, pool)
	if err != nil {
		log.Fatalf("❌ Ошибка миграции: %v", err)
	}

	// Добавляем аккаунты с транзакцией и валидацией
	accounts := []Account{
		{"alice", "alice@example.com"},
		{"bob", "bob@example.com"},
		//{"x", "bad-example.com"}
	}

	err = createAccountsInTransaction(ctx, pool, accounts)
	if err != nil {
		log.Fatalf("❌ Ошибка создания аккаунтов: %v", err)
	}

	// Выводим результат
	printAccountsAndRoles(ctx, pool)
	fmt.Println("\nВсё успешно!")
}

// Применение миграции из embed-файла
func applyMigration(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, migrationSQL)
	if err != nil {
		return fmt.Errorf("ошибка выполнения миграции: %w", err)
	}
	fmt.Println("Миграция применена")
	return nil
}

// Создание аккаунтов в одной транзакции
func createAccountsInTransaction(ctx context.Context, pool *pgxpool.Pool, accounts []Account) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("не удалось начать транзакцию: %w", err)
	}
	defer tx.Rollback(ctx) // откатываем, если не сделаем Commit

	// Вставляем роли (если их нет)
	roles := []string{"user", "admin"}
	for _, role := range roles {
		_, err = tx.Exec(ctx, "INSERT INTO roles (name) VALUES ($1) ON CONFLICT (name) DO NOTHING", role)
		if err != nil {
			return fmt.Errorf("ошибка вставки роли %s: %w", role, err)
		}
	}

	// Валидация и вставка аккаунтов
	for _, acc := range accounts {
		if err := acc.Validate(); err != nil {
			return fmt.Errorf("валидация не пройдена для %s: %w", acc.Username, err)
		}

		var accountID int
		err := tx.QueryRow(ctx,
			"INSERT INTO accounts (username, email) VALUES ($1, $2) RETURNING id",
			acc.Username, acc.Email,
		).Scan(&accountID)
		if err != nil {
			// pgx возвращает ошибку, если нарушено ограничение (например, дубль email)
			return fmt.Errorf("ошибка вставки аккаунта %s: %w", acc.Username, err)
		}

		// Назначаем роль "user"
		_, err = tx.Exec(ctx,
			"INSERT INTO account_roles (account_id, role_id) SELECT $1, id FROM roles WHERE name = 'user'",
			accountID,
		)
		if err != nil {
			return fmt.Errorf("ошибка назначения роли для %s: %w", acc.Username, err)
		}

		// Создаём сессию
		sessionID := uuid.New()
		expiresAt := time.Now().Add(24 * time.Hour)
		_, err = tx.Exec(ctx,
			"INSERT INTO sessions (id, account_id, token_hash, expires_at) VALUES ($1, $2, $3, $4)",
			sessionID, accountID, "sha256:"+sessionID.String(), expiresAt,
		)
		if err != nil {
			return fmt.Errorf("ошибка создания сессии для %s: %w", acc.Username, err)
		}
	}

	// Коммитим транзакцию
	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	fmt.Println("Аккаунты успешно созданы в транзакции")
	return nil
}

// Вывод данных
func printAccountsAndRoles(ctx context.Context, pool *pgxpool.Pool) {
	rows, err := pool.Query(ctx, `
		SELECT a.username, a.email, r.name AS role, s.expires_at
		FROM accounts a
		JOIN account_roles ar ON a.id = ar.account_id
		JOIN roles r ON ar.role_id = r.id
		JOIN sessions s ON s.account_id = a.id
		ORDER BY a.username;
	`)
	if err != nil {
		log.Printf("  Ошибка вывода: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("\n  Аккаунты, роли и сессии:")
	for rows.Next() {
		var username, email, role string
		var expires time.Time
		err = rows.Scan(&username, &email, &role, &expires)
		if err != nil {
			log.Printf("Ошибка чтения строки: %v", err)
			continue
		}
		fmt.Printf("  %s (%s)  - роль: %s, сессия до: %s\n",
			username, email, role, expires.Format("2006-01-02 15:04"))
	}
}
