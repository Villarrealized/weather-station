package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const port string = "8367"
const minTempF float32 = -67.0
const maxTempF float32 = 257.0

type TempSensorData struct {
	MacAddress  string  `json:"mac"`
	Temperature float32 `json:"temperature"`
}

type Device struct {
	ID   string
	Name string
}

type TempSensorReading struct {
	Temperature float32 `json:"temperature"`
	Timestamp   string  `json:"timestamp"`
}

var (
	devicesCache    map[string]Device
	currentReadings map[string]TempSensorReading
	mu              sync.Mutex
)

func createTables(db *sql.DB) {
	var statements = []string{
		"CREATE TABLE IF NOT EXISTS devices(id TEXT NOT NULL PRIMARY KEY, name TEXT)",
		"CREATE TABLE IF NOT EXISTS temperature_readings(id INTEGER PRIMARY KEY, device_id TEXT NOT NULL, temp_f FLOAT, timestamp DATETIME)",
	}

	for _, stmt := range statements {
		_, err := db.Exec(stmt)
		if err != nil {
			log.Fatalf("%q: %s\n", err, stmt)
		}
	}
}

func getDevice(id string, db *sql.DB) Device {
	device, ok := devicesCache[id]
	if ok {
		return device
	}

	row := db.QueryRow("select id, name from devices where id = ?", id)

	if err := row.Scan(&device.ID, &device.Name); err != nil {
		log.Fatal(err)
	}
	devicesCache[id] = device

	return device
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	db, err := sql.Open("sqlite3", "./data/weather-station.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	createTables(db)

	currentReadings = make(map[string]TempSensorReading)
	location, err := time.LoadLocation("America/Denver")
	if err != nil {
		slog.Error("failed to get location", "error", err)
		panic("location fetching failed")
	}

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		readings, err := json.Marshal(currentReadings)
		if err != nil {
			slog.Error("failed to marshall temperature data", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write(readings)
	})

	r.Post("/temperature", func(w http.ResponseWriter, r *http.Request) {
		var data TempSensorData
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			slog.Error("failed to decode temperature data", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		device := getDevice(data.MacAddress, db)

		// sometimes the temp can be outside the valid range
		// we will log an error and ignore it as a bad read
		if data.Temperature < minTempF || data.Temperature > maxTempF {
			slog.Error("temperature outside of valid range", "device", device.Name, "temp", data.Temperature)
			w.WriteHeader(http.StatusOK)
			return
		}

		mu.Lock()
		defer mu.Unlock()
		now := time.Now().In(location)
		currentReadings[device.Name] = TempSensorReading{Temperature: data.Temperature, Timestamp: now.Format(time.RFC3339)}

		slog.Info("New reading", "device", device.Name, "temp", data.Temperature)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("healthy"))
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 Not Found"))
	})

	fmt.Println("Listening on", "port", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), r); err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}
