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
	defer log.Println("=== ЗАВЕРШЕНИО ОБРАБОТКИ БРОНИРОВАНИЯ ===")

	if r.Method != http.MethodPost {
		log.Println("Ошибка: требуется POST-запрос")
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Ошибка парсинга формы: %v", err)
		http.Error(w, "Ошибка обработки формы", http.StatusBadRequest)
		return
	}

	tableID, err := strconv.Atoi(r.FormValue("table_id"))
	if err != nil {
		log.Printf("Ошибка парсинга table_id: %v", err)
		http.Error(w, "Неверный ID столика", http.StatusBadRequest)
		return
	}

	guests, err := strconv.Atoi(r.FormValue("guests"))
	if err != nil {
		log.Printf("Ошибка парсинга guests: %v", err)
		http.Error(w, "Неверное количество гостей", http.StatusBadRequest)
		return
	}

	dateStr := r.FormValue("date")
	parsedDate, err := time.Parse("2006-01-02T15:04", dateStr)
	if err != nil {
		log.Printf("Ошибка парсинга даты: %v", err)
		http.Error(w, "Неверный формат даты", http.StatusBadRequest)
		return
	}

	if parsedDate.Before(time.Now()) {
		log.Println("Ошибка: дата в прошлом")
		http.Error(w, "Дата должна быть в будущем", http.StatusBadRequest)
		return
	}

	// Проверка доступности столика
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

	// Создание и сохранение бронирования
	booking := &models.Booking{
		TableID: tableID,
		Name:    r.FormValue("name"),
		Email:   r.FormValue("email"),
		Phone:   r.FormValue("phone"),
		Date:    dateStr,
		Guests:  guests,
	}

	_, err = models.DB.NamedExec(`
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

	http.Redirect(w, r, "/bookings?success=true", http.StatusSeeOther)
}

// bookingsHandler - страница списка бронирований
func bookingsHandler(w http.ResponseWriter, r *http.Request) {
	success := r.URL.Query().Get("success") == "true"

	type BookingView struct {
		models.Booking
		FormattedDate string
		FormattedTime string
	}

	var bookings []BookingView

	err := models.DB.Select(&bookings, `
		SELECT b.*, t.capacity, t.location 
		FROM bookings b JOIN tables t ON b.table_id = t.id
		ORDER BY b.date DESC`)

	if err != nil {
		renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for i := range bookings {
		date, err := time.Parse(time.RFC3339, bookings[i].Date)
		if err != nil {
			bookings[i].FormattedDate = bookings[i].Date
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

// renderError - вспомогательная функция для отображения ошибок
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
