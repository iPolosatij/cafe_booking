package models

import (
	"time"

	_ "github.com/lib/pq"
)

type Table struct {
	ID       int    `db:"id"`
	Capacity int    `db:"capacity"`
	Location string `db:"location"`
}

type Booking struct {
	ID        int       `db:"id"`
	TableID   int       `db:"table_id"`
	Name      string    `db:"name"`
	Email     string    `db:"email"`
	Phone     string    `db:"phone"`
	Date      string    `db:"date"` // Оставляем как string
	Guests    int       `db:"guests"`
	CreatedAt time.Time `db:"created_at"`
	Capacity  int       `db:"capacity"`
	Location  string    `db:"location"`
}
