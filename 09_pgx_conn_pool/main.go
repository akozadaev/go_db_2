package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const connStr = "postgres://ibs:ibs@localhost:5432/ibs_pool?sslmode=disable"

func main() {
	ctx := context.Background()

	// Создаём пул подключений
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("Не удалось создать пул подключений: %v", err)
	}
	defer pool.Close()

	fmt.Println("Пул подключений к PostgreSQL создан.")

	// Выполняем миграцию (создание таблиц)
	err = runMigration(ctx, pool)
	if err != nil {
		log.Fatalf("Ошибка миграции: %v", err)
	}

	// Заполняем данными
	err = seedData(ctx, pool)
	if err != nil {
		log.Fatalf("Ошибка заполнения данных: %v", err)
	}

	// Выводим данные
	err = printData(ctx, pool)
	if err != nil {
		log.Fatalf("Ошибка вывода данных: %v", err)
	}

	fmt.Println("\n Пример завершён успешно!")
}

// === МИГРАЦИЯ ===
func runMigration(ctx context.Context, pool *pgxpool.Pool) error {
	tables := []string{
		`
		CREATE TABLE IF NOT EXISTS accounts (
			id SERIAL PRIMARY KEY,
			username VARCHAR(50) UNIQUE NOT NULL,
			email VARCHAR(100) UNIQUE NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS roles (
			id SERIAL PRIMARY KEY,
			name VARCHAR(50) UNIQUE NOT NULL
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS account_roles (
			account_id INTEGER REFERENCES accounts(id) ON DELETE CASCADE,
			role_id INTEGER REFERENCES roles(id) ON DELETE CASCADE,
			PRIMARY KEY (account_id, role_id)
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS permissions (
			id SERIAL PRIMARY KEY,
			name VARCHAR(50) UNIQUE NOT NULL
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS role_permissions (
			role_id INTEGER REFERENCES roles(id) ON DELETE CASCADE,
			permission_id INTEGER REFERENCES permissions(id) ON DELETE CASCADE,
			PRIMARY KEY (role_id, permission_id)
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
		`,
	}

	for _, query := range tables {
		_, err := pool.Exec(ctx, query)
		if err != nil {
			return fmt.Errorf("ошибка при создании таблицы: %w", err)
		}
	}
	fmt.Println("Все таблицы созданы.")
	return nil
}

// === ЗАПОЛНЕНИЕ ДАННЫМИ ===
func seedData(ctx context.Context, pool *pgxpool.Pool) error {
	// Роли
	roles := []string{"admin", "user", "moderator"}
	for _, r := range roles {
		_, err := pool.Exec(ctx, "INSERT INTO roles (name) VALUES ($1) ON CONFLICT (name) DO NOTHING;", r)
		if err != nil {
			return err
		}
	}

	// Права
	permissions := []string{"read", "write", "delete", "manage_users"}
	for _, p := range permissions {
		_, err := pool.Exec(ctx, "INSERT INTO permissions (name) VALUES ($1) ON CONFLICT (name) DO NOTHING;", p)
		if err != nil {
			return err
		}
	}

	// Аккаунты
	accounts := [][2]string{
		{"alice", "alice@example.com"},
		{"bob", "bob@example.com"},
		{"charlie", "charlie@example.com"},
	}
	for _, acc := range accounts {
		_, err := pool.Exec(ctx, `
			INSERT INTO accounts (username, email) 
			VALUES ($1, $2) 
			ON CONFLICT (email) DO NOTHING;
		`, acc[0], acc[1])
		if err != nil {
			return err
		}
	}

	// Связи аккаунт-роль
	// alice - admin, user
	// bob - user
	// charlie - moderator
	_, err := pool.Exec(ctx, `
		INSERT INTO account_roles (account_id, role_id)
		SELECT a.id, r.id
		FROM accounts a, roles r
		WHERE a.username = 'alice' AND r.name IN ('admin', 'user')
		ON CONFLICT DO NOTHING;
	`)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO account_roles (account_id, role_id)
		SELECT a.id, r.id
		FROM accounts a, roles r
		WHERE a.username = 'bob' AND r.name = 'user'
		ON CONFLICT DO NOTHING;
	`)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO account_roles (account_id, role_id)
		SELECT a.id, r.id
		FROM accounts a, roles r
		WHERE a.username = 'charlie' AND r.name = 'moderator'
		ON CONFLICT DO NOTHING;
	`)
	if err != nil {
		return err
	}

	// Связи роль-право
	// admin  - все права
	// user  - read, write
	// moderator  - read, write, delete
	_, err = pool.Exec(ctx, `
		INSERT INTO role_permissions (role_id, permission_id)
		SELECT r.id, p.id
		FROM roles r, permissions p
		WHERE r.name = 'admin'
		ON CONFLICT DO NOTHING;
	`)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO role_permissions (role_id, permission_id)
		SELECT r.id, p.id
		FROM roles r, permissions p
		WHERE r.name = 'user' AND p.name IN ('read', 'write')
		ON CONFLICT DO NOTHING;
	`)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO role_permissions (role_id, permission_id)
		SELECT r.id, p.id
		FROM roles r, permissions p
		WHERE r.name = 'moderator' AND p.name IN ('read', 'write', 'delete')
		ON CONFLICT DO NOTHING;
	`)
	if err != nil {
		return err
	}

	// Сессии
	now := time.Now()
	expires := now.Add(24 * time.Hour)
	_, err = pool.Exec(ctx, `
		INSERT INTO sessions (account_id, token_hash, expires_at)
		SELECT id, 'hash123', $1 FROM accounts WHERE username = 'alice'
		ON CONFLICT DO NOTHING;
	`, expires)
	if err != nil {
		return err
	}

	fmt.Println("Тестовые данные добавлены.")
	return nil
}

// === ВЫВОД ДАННЫХ ===
func printData(ctx context.Context, pool *pgxpool.Pool) error {
	// Аккаунты с ролями
	rows, err := pool.Query(ctx, `
		SELECT a.username, r.name AS role
		FROM accounts a
		JOIN account_roles ar ON a.id = ar.account_id
		JOIN roles r ON ar.role_id = r.id
		ORDER BY a.username, r.name;
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Println("\n Аккаунты и их роли:")
	for rows.Next() {
		var username, role string
		err = rows.Scan(&username, &role)
		if err != nil {
			return err
		}
		fmt.Printf("  %s  - %s\n", username, role)
	}

	// Роли и их права
	rows2, err := pool.Query(ctx, `
		SELECT r.name AS role, p.name AS permission
		FROM roles r
		JOIN role_permissions rp ON r.id = rp.role_id
		JOIN permissions p ON rp.permission_id = p.id
		ORDER BY r.name, p.name;
	`)
	if err != nil {
		return err
	}
	defer rows2.Close()

	fmt.Println("\n Роли и их права:")
	for rows2.Next() {
		var role, permission string
		err := rows2.Scan(&role, &permission)
		if err != nil {
			return err
		}
		fmt.Printf("  %s  - %s\n", role, permission)
	}

	// Сессии
	rows3, err := pool.Query(ctx, `
		SELECT a.username, s.expires_at
		FROM sessions s
		JOIN accounts a ON s.account_id = a.id;
	`)
	if err != nil {
		return err
	}
	defer rows3.Close()

	fmt.Println("\n Активные сессии:")
	for rows3.Next() {
		var username string
		var expires time.Time
		err := rows3.Scan(&username, &expires)
		if err != nil {
			return err
		}
		fmt.Printf("  %s - истекает %s\n", username, expires.Format("2006-01-02 15:04"))
	}

	return nil
}
