package models

import (
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

var DB *sqlx.DB

func InitDB(connectionString string) error {
	log.Println("Попытка подключения со строкой:", connectionString)

	// Проверяем драйвер
	drivers := sql.Drivers()
	log.Println("Доступные драйверы:", drivers)

	var err error
	DB, err = sqlx.Open("postgres", connectionString)
	if err != nil {
		log.Printf("FATAL: sqlx.Open error: %v", err)
		return err
	}

	// Настройки пула соединений
	DB.SetMaxOpenConns(5)
	DB.SetMaxIdleConns(2)
	DB.SetConnMaxLifetime(30 * time.Minute)

	// Проверяем подключение
	log.Println("Выполняем Ping...")
	err = DB.Ping()
	if err != nil {
		log.Printf("FATAL: Ping error: %v", err)

		// Дополнительная диагностика
		var pgErr *pq.Error
		if errors.As(err, &pgErr) {
			log.Println("PostgreSQL error details:")
			log.Println("Code:", pgErr.Code)
			log.Println("Message:", pgErr.Message)
			log.Println("Detail:", pgErr.Detail)
		}

		return err
	}

	log.Println("Успешное подключение к PostgreSQL!")

	// Проверяем версию
	var version string
	err = DB.Get(&version, "SELECT version()")
	if err != nil {
		log.Printf("WARN: Не удалось получить версию: %v", err)
	} else {
		log.Println("Версия PostgreSQL:", version)
	}

	return nil
}

func CreateTables() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS tables (
			id SERIAL PRIMARY KEY,
			capacity INTEGER NOT NULL,
			location TEXT NOT NULL
		);
		
		CREATE TABLE IF NOT EXISTS bookings (
			id SERIAL PRIMARY KEY,
			table_id INTEGER REFERENCES tables(id),
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			phone TEXT NOT NULL,
			date TIMESTAMP NOT NULL,
			guests INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

func SeedData() error {
	_, err := DB.Exec(`
		INSERT INTO tables (capacity, location) VALUES
		(2, 'У окна'),
		(4, 'В центре'),
		(6, 'VIP зона')
		ON CONFLICT DO NOTHING;
	`)
	return err
}

func Close() error {
	return DB.Close()
}
