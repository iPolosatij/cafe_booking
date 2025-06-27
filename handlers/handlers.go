package handlers

import (
	"cafe-booking/models"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
)

func InitRoutes() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/book", bookHandler)
	http.HandleFunc("/bookings", bookingsHandler)
	http.HandleFunc("/book/submit", submitBookingHandler)
}

// renderTemplate - универсальная функция рендеринга шаблонов
func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	templates := template.Must(template.ParseFiles(
		"templates/base.html",
		"templates/"+tmpl+".html",
	))

	err := templates.ExecuteTemplate(w, "base", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// homeHandler - главная страница со списком столиков
func homeHandler(w http.ResponseWriter, r *http.Request) {
	var tables []models.Table
	err := models.DB.Select(&tables, "SELECT * FROM tables ORDER BY id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Title  string
		Tables []models.Table
	}{
		Title:  "Бронирование столиков",
		Tables: tables,
	}

	renderTemplate(w, "home", data)
}

// bookHandler - страница бронирования конкретного столика
func bookHandler(w http.ResponseWriter, r *http.Request) {
	tableID, err := strconv.Atoi(r.URL.Query().Get("table_id"))
	if err != nil {
		renderError(w, "Неверный ID столика", http.StatusBadRequest)
		return
	}

	var table models.Table
	err = models.DB.Get(&table, "SELECT * FROM tables WHERE id = $1", tableID)
	if err != nil {
		renderError(w, "Столик не найден", http.StatusNotFound)
		return
	}

	data := struct {
		Title string
		Table models.Table
	}{
		Title: fmt.Sprintf("Бронирование столика #%d", table.ID),
		Table: table,
	}

	renderTemplate(w, "book", data)
}

// submitBookingHandler - обработчик отправки формы бронирования
func submitBookingHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n=== НАЧАЛО ОБРАБОТКИ БРОНИРОВАНИЯ ===")
	defer log.Println("=== ЗАВЕРШЕНИЕ ОБРАБОТКИ БРОНИРОВАНИЯ ===")

	// 1. Логируем входящий запрос
	log.Printf("Метод: %s, URL: %s", r.Method, r.URL.Path)
	log.Println("Заголовки:", r.Header)

	// 2. Проверка метода
	if r.Method != http.MethodPost {
		log.Println("Ошибка: требуется POST-запрос")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	// 3. Парсинг формы с логированием
	log.Println("Парсинг формы...")
	if err := r.ParseForm(); err != nil {
		log.Printf("Ошибка парсинга формы: %v", err)
		http.Error(w, "Ошибка обработки формы", http.StatusBadRequest)
		return
	}
	log.Println("Данные формы:", r.Form)

	// 4. Валидация и логирование полей
	tableID, err := strconv.Atoi(r.FormValue("table_id"))
	if err != nil {
		log.Printf("Ошибка парсинга table_id: %v, значение: %s", err, r.FormValue("table_id"))
		http.Error(w, "Неверный ID столика", http.StatusBadRequest)
		return
	}
	log.Printf("TableID: %d", tableID)

	guests, err := strconv.Atoi(r.FormValue("guests"))
	if err != nil {
		log.Printf("Ошибка парсинга guests: %v, значение: %s", err, r.FormValue("guests"))
		http.Error(w, "Неверное количество гостей", http.StatusBadRequest)
		return
	}
	log.Printf("Guests: %d", guests)

	dateStr := r.FormValue("date")
	log.Printf("Raw date from form: %s", dateStr)

	parsedDate, err := time.Parse("2006-01-02T15:04", dateStr)
	if err != nil {
		log.Printf("Ошибка парсинга даты: %v", err)
		http.Error(w, "Неверный формат даты. Используйте YYYY-MM-DDTHH:MM", http.StatusBadRequest)
		return
	}
	log.Printf("Parsed date: %v", parsedDate)

	if parsedDate.Before(time.Now()) {
		log.Println("Ошибка: дата в прошлом")
		http.Error(w, "Дата должна быть в будущем", http.StatusBadRequest)
		return
	}

	// 5. Проверка доступности столика
	log.Println("Проверка доступности столика...")
	var existingBooking int
	err = models.DB.Get(&existingBooking,
		"SELECT COUNT(*) FROM bookings WHERE table_id = $1 AND date = $2",
		tableID, dateStr)

	if err != nil {
		log.Printf("Ошибка проверки брони: %v", err)
		http.Error(w, "Ошибка проверки доступности", http.StatusInternalServerError)
		return
	}

	if existingBooking > 0 {
		log.Println("Столик уже занят")
		http.Error(w, "Этот столик уже забронирован на выбранное время", http.StatusConflict)
		return
	}

	// 6. Создание объекта бронирования
	booking := &models.Booking{
		TableID: tableID,
		Name:    r.FormValue("name"),
		Email:   r.FormValue("email"),
		Phone:   r.FormValue("phone"),
		Date:    dateStr,
		Guests:  guests,
	}
	log.Printf("Создан объект бронирования: %+v", booking)

	// 7. Сохранение в БД
	log.Println("Сохранение в БД...")
	result, err := models.DB.NamedExec(`
        INSERT INTO bookings 
        (table_id, name, email, phone, date, guests)
        VALUES 
        (:table_id, :name, :email, :phone, :date, :guests)`,
		booking)

	if err != nil {
		log.Printf("Ошибка сохранения брони: %v", err)
		http.Error(w, "Ошибка сохранения бронирования", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("Успешно сохранено. Затронуто строк: %d", rowsAffected)

	// 8. Перенаправление
	log.Println("Перенаправление на /bookings")
	http.Redirect(w, r, "/bookings?success=true", http.StatusSeeOther)
}

// bookingsHandler - страница списка бронирований
func bookingsHandler(w http.ResponseWriter, r *http.Request) {
	success := r.URL.Query().Get("success") == "true"

	// Создаем структуру для хранения данных бронирования с форматированной датой
	type BookingView struct {
		models.Booking
		FormattedDate string
		FormattedTime string
	}

	var bookings []BookingView

	// Получаем данные из БД
	err := models.DB.Select(&bookings, `
        SELECT b.*, t.capacity, t.location 
        FROM bookings b JOIN tables t ON b.table_id = t.id
        ORDER BY b.date DESC`)

	if err != nil {
		renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Форматируем дату для каждого бронирования
	for i := range bookings {
		date, err := time.Parse(time.RFC3339, bookings[i].Date)
		if err != nil {
			log.Printf("Ошибка парсинга даты: %v", err)
			bookings[i].FormattedDate = bookings[i].Date // Оставляем оригинальное значение
			bookings[i].FormattedTime = ""
		} else {
			bookings[i].FormattedDate = date.Format("02.01.2006")
			bookings[i].FormattedTime = date.Format("15:04")
		}
	}

	data := struct {
		Title    string
		Bookings []BookingView
		Success  bool
	}{
		Title:    "Мои бронирования",
		Bookings: bookings,
		Success:  success,
	}

	renderTemplate(w, "bookings", data)
}

// Вспомогательные функции

func renderError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	tmpl := template.Must(template.ParseFiles(
		"templates/base.html",
		"templates/error.html",
	))
	tmpl.ExecuteTemplate(w, "base", map[string]string{
		"Title":   "Ошибка",
		"Message": message,
	})
}

func parseBookingForm(r *http.Request) (*models.Booking, error) {
	tableID, err := strconv.Atoi(r.FormValue("table_id"))
	if err != nil {
		return nil, fmt.Errorf("Неверный ID столика")
	}

	guests, err := strconv.Atoi(r.FormValue("guests"))
	if err != nil {
		return nil, fmt.Errorf("Неверное количество гостей")
	}

	date, err := time.Parse("2006-01-02T15:04", r.FormValue("date"))
	if err != nil {
		return nil, fmt.Errorf("Неверный формат даты")
	}

	return &models.Booking{
		TableID: tableID,
		Name:    r.FormValue("name"),
		Email:   r.FormValue("email"),
		Phone:   r.FormValue("phone"),
		Date:    date.Format(time.RFC3339),
		Guests:  guests,
	}, nil
}

func isTableBooked(tableID int, date string) bool {
	var count int
	err := models.DB.Get(&count, `
		SELECT COUNT(*) FROM bookings 
		WHERE table_id = $1 AND date = $2`,
		tableID, date)
	return err == nil && count > 0
}

func saveBooking(booking *models.Booking) error {
	_, err := models.DB.NamedExec(`
		INSERT INTO bookings 
		(table_id, name, email, phone, date, guests)
		VALUES 
		(:table_id, :name, :email, :phone, :date, :guests)`,
		booking)
	return err
}
