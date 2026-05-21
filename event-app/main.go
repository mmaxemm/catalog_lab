package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

var db *sql.DB

const initSQL = `
CREATE TABLE IF NOT EXISTS place (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    capacity    INT NOT NULL CHECK (capacity > 0),
    address     TEXT NOT NULL DEFAULT '',
    opening_date DATE NOT NULL,
    area        NUMERIC(12, 2) NOT NULL CHECK (area > 0),
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMP DEFAULT NULL
);

CREATE TABLE IF NOT EXISTS event (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    start_time  TIMESTAMP NOT NULL,
    end_time    TIMESTAMP NOT NULL,
    price       NUMERIC(12, 2) NOT NULL CHECK (price >= 0) DEFAULT 0,
    age_limit   INT NOT NULL CHECK (age_limit >= 0) DEFAULT 0,
    place_id    BIGINT NOT NULL REFERENCES place(id),
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMP DEFAULT NULL,
    CONSTRAINT check_dates CHECK (start_time <= end_time)
);`

type Place struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Capacity    int     `json:"capacity"`
	Address     string  `json:"address"`
	OpeningDate string  `json:"opening_date"`
	Area        float64 `json:"area"`
}

type Event struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	StartTime   string  `json:"start_time"`
	EndTime     string  `json:"end_time"`
	Price       float64 `json:"price"`
	AgeLimit    int     `json:"age_limit"`
	PlaceID     int     `json:"place_id"`
	PlaceName   string  `json:"place_name,omitempty"`
}

func main() {
	var err error
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/event_db?sslmode=disable"
	}

	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Ошибка конфигурации БД:", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal("Не удалось подключиться к БД:", err)
	}

	_, err = db.Exec(initSQL)
	if err != nil {
		log.Fatal("Ошибка инициализации таблиц в БД:", err)
	}
	log.Println("База данных успешно проверена и готова к работе.")

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	mux.HandleFunc("GET /api/places", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, name, capacity, address, to_char(opening_date, 'YYYY-MM-DD'), area FROM place WHERE deleted_at IS NULL")
		if err != nil {
			http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var places []Place = []Place{}
		for rows.Next() {
			var p Place
			rows.Scan(&p.ID, &p.Name, &p.Capacity, &p.Address, &p.OpeningDate, &p.Area)
			places = append(places, p)
		}
		json.NewEncoder(w).Encode(places)
	})

	mux.HandleFunc("POST /api/places", func(w http.ResponseWriter, r *http.Request) {
		var p Place
		json.NewDecoder(r.Body).Decode(&p)
		_, err := db.Exec("INSERT INTO place (name, capacity, address, opening_date, area) VALUES ($1, $2, $3, $4, $5)",
			p.Name, p.Capacity, p.Address, p.OpeningDate, p.Area)
		if err != nil {
			http.Error(w, "Неправильно введены данные", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("PUT /api/places/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var p Place
		json.NewDecoder(r.Body).Decode(&p)
		_, err := db.Exec("UPDATE place SET name=$1, capacity=$2, address=$3, opening_date=$4, area=$5, updated_at=NOW() WHERE id=$6",
			p.Name, p.Capacity, p.Address, p.OpeningDate, p.Area, id)
		if err != nil {
			http.Error(w, "Неправильно введены данные", http.StatusBadRequest)
			return
		}
	})

	mux.HandleFunc("DELETE /api/places/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		_, err := db.Exec("UPDATE place SET deleted_at=NOW() WHERE id=$1", id)
		if err != nil {
			http.Error(w, "Ошибка удаления", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("GET /api/events", func(w http.ResponseWriter, r *http.Request) {
		query := `SELECT e.id, e.name, e.description, to_char(e.start_time, 'YYYY-MM-DD"T"HH24:MI'), 
                  to_char(e.end_time, 'YYYY-MM-DD"T"HH24:MI'), e.price, e.age_limit, e.place_id, p.name 
                  FROM event e JOIN place p ON e.place_id = p.id WHERE e.deleted_at IS NULL`
		rows, err := db.Query(query)
		if err != nil {
			http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var events []Event = []Event{}
		for rows.Next() {
			var e Event
			rows.Scan(&e.ID, &e.Name, &e.Description, &e.StartTime, &e.EndTime, &e.Price, &e.AgeLimit, &e.PlaceID, &e.PlaceName)
			events = append(events, e)
		}
		json.NewEncoder(w).Encode(events)
	})

	mux.HandleFunc("POST /api/events", func(w http.ResponseWriter, r *http.Request) {
		var e Event
		json.NewDecoder(r.Body).Decode(&e)
		_, err := db.Exec("INSERT INTO event (name, description, start_time, end_time, price, age_limit, place_id) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			e.Name, e.Description, e.StartTime, e.EndTime, e.Price, e.AgeLimit, e.PlaceID)
		if err != nil {
			http.Error(w, "Неправильно введены данные", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("PUT /api/events/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var e Event
		json.NewDecoder(r.Body).Decode(&e)
		_, err := db.Exec("UPDATE event SET name=$1, description=$2, start_time=$3, end_time=$4, price=$5, age_limit=$6, place_id=$7, updated_at=NOW() WHERE id=$8",
			e.Name, e.Description, e.StartTime, e.EndTime, e.Price, e.AgeLimit, e.PlaceID, id)
		if err != nil {
			http.Error(w, "Неправильно введены данные", http.StatusBadRequest)
			return
		}
	})

	mux.HandleFunc("DELETE /api/events/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		_, err := db.Exec("UPDATE event SET deleted_at=NOW() WHERE id=$1", id)
		if err != nil {
			http.Error(w, "Ошибка удаления", http.StatusInternalServerError)
		}
	})

	log.Println("Приложение запущено на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
