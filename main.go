package main

import (
	"cafe-booking/config"
	"cafe-booking/handlers"
	"cafe-booking/models"
	"log"
	"net/http"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	// Загрузка конфигурации
	cfg := config.GetConfig()

	testConnection()

	// Инициализация БД
	err := models.InitDB(cfg.GetDBConnectionString())
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}
	defer models.Close()

	// Создание таблиц
	err = models.CreateTables()
	if err != nil {
		log.Fatalf("Ошибка создания таблиц: %v", err)
	}

	// Заполнение начальных данных
	err = models.SeedData()
	if err != nil {
		log.Printf("Ошибка заполнения данных: %v", err)
	}

	// Настройка маршрутов
	handlers.InitRoutes()

	// Обработка статических файлов
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Запуск сервера
	port := cfg.Server.Port
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	log.Printf("Сервер запущен на http://localhost:%s", port)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func testConnection() {
	connStr := "host=localhost port=5432 user=cafe_user password=yourpassword dbname=cafe_booking sslmode=disable"
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Fatalf("Тестовое подключение не удалось: %v", err)
	}
	log.Println("Тестовое подключение успешно!")
	db.Close()
}
