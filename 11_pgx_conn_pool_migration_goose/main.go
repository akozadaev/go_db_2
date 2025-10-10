package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "ibs"
	password = "ibs"
	dbname   = "ibs_goose"
)

// Account представляет аккаунт
type Account struct {
	Username string
	Email    string
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func (a *Account) Validate() error {
	if len(a.Username) < 3 {
		return fmt.Errorf("username must be at least 3 characters")
	}
	if len(a.Username) > 50 {
		return fmt.Errorf("username too long")
	}
	if !emailRegex.MatchString(a.Email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func main() {
	ctx := context.Background()

	// 1. Создаём пул для приложения
	pool, err := pgxpool.New(ctx, fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", user, password, host, port, dbname))
	if err != nil {
		log.Fatalf("❌ Не удалось создать пул: %v", err)
	}
	defer pool.Close()

	// 2. Применяем миграции через goose
	err = applyMigrationsWithGoose()
	if err != nil {
		log.Fatalf("❌ Ошибка миграций: %v", err)
	}

	// 3. Создаём аккаунты в транзакции
	accounts := []Account{
		{"alice", "alice@example.com"},
		{"bob", "bob@example.com"},
	}

	err = createAccountsInTransaction(ctx, pool, accounts)
	if err != nil {
		log.Fatalf("❌ Ошибка создания аккаунтов: %v", err)
	}

	// 4. Выводим данные
	printAccountsAndRoles(ctx, pool)
	fmt.Println("\nГотово!")
}

// Применение миграций через goose
func applyMigrationsWithGoose() error {
	// Goose работает через стандартный database/sql, поэтому нужен stdlib-адаптер
	config, err := pgx.ParseConfig(fmt.Sprintf("host=%s port=%d user=%s password=%s database=%s sslmode=disable",
		host, port, user, password, dbname))
	if err != nil {
		return fmt.Errorf("ошибка парсинга конфига: %w", err)
	}

	// Создаём *sql.DB через адаптер pgx  database/sql
	db := stdlib.OpenDB(*config)
	defer db.Close()

	// Применяем все миграции вверх
	if err = goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("goose.Up failed: %w", err)
	}

	fmt.Println("Миграции применены через goose")
	return nil
}

// Создание аккаунтов в транзакции (как в прошлом примере)
func createAccountsInTransaction(ctx context.Context, pool *pgxpool.Pool, accounts []Account) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("не удалось начать транзакцию: %w", err)
	}
	defer tx.Rollback(ctx)

	// Убедимся, что роли существуют
	roles := []string{"user", "admin"}
	for _, role := range roles {
		_, err = tx.Exec(ctx, "INSERT INTO roles (name) VALUES ($1) ON CONFLICT (name) DO NOTHING", role)
		if err != nil {
			return fmt.Errorf("ошибка вставки роли %s: %w", role, err)
		}
	}

	for _, acc := range accounts {
		if err := acc.Validate(); err != nil {
			return fmt.Errorf("валидация %s: %w", acc.Username, err)
		}

		var accountID int
		err = tx.QueryRow(ctx,
			"INSERT INTO accounts (username, email) VALUES ($1, $2) ON CONFLICT (email) DO NOTHING RETURNING id",
			acc.Username, acc.Email,
		).Scan(&accountID)
		if err != nil {
			if err == pgx.ErrNoRows {
				// Уже существует — пропускаем
				continue
			}
			return fmt.Errorf("ошибка вставки аккаунта %s: %w", acc.Username, err)
		}

		_, err = tx.Exec(ctx,
			"INSERT INTO account_roles (account_id, role_id) SELECT $1, id FROM roles WHERE name = 'user' ON CONFLICT DO NOTHING",
			accountID,
		)
		if err != nil {
			return fmt.Errorf("ошибка назначения роли: %w", err)
		}

		sessionID := uuid.New()
		expiresAt := time.Now().Add(24 * time.Hour)
		_, err = tx.Exec(ctx,
			"INSERT INTO sessions (id, account_id, token_hash, expires_at) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING",
			sessionID, accountID, "sha256:"+sessionID.String(), expiresAt,
		)
		if err != nil {
			return fmt.Errorf("ошибка сессии: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("ошибка коммита: %w", err)
	}

	fmt.Println("Аккаунты созданы в транзакции")
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
		log.Printf("⚠️ Ошибка запроса: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("\nАккаунты:")
	for rows.Next() {
		var username, email, role string
		var expires time.Time
		_ = rows.Scan(&username, &email, &role, &expires)
		fmt.Printf("  %s (%s)  %s, сессия до %s\n", username, email, role, expires.Format("2006-01-02"))
	}
}
