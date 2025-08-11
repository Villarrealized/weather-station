package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"database/sql"

	"github.com/Villarrealized/weather-station/data"
	_ "github.com/mattn/go-sqlite3"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const port string = "8367"
const minTempF float32 = -67.0
const maxTempF float32 = 257.0

type TemperatureRequest struct {
	MacAddress string  `json:"mac"`
	TempF      float32 `json:"temperature"`
}

var db *data.Database

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	dbConnection, err := sql.Open("sqlite3", "./data/weather-station.db")
	if err != nil {
		log.Fatal(err)
	}
	defer dbConnection.Close()

	db = data.NewDatabase(dbConnection)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		readings, err := json.Marshal(db.GetTemperatureReadings())
		if err != nil {
			slog.Error("failed to marshall temperature data", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write(readings)
	})

	r.Post("/temperature", func(w http.ResponseWriter, r *http.Request) {
		var data TemperatureRequest
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			slog.Error("failed to decode temperature request", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		device := db.GetDevice(data.MacAddress)

		// sometimes the temp can be outside the valid range
		// we will log an error and ignore it as a bad read
		if data.TempF < minTempF || data.TempF > maxTempF {
			slog.Error("temperature outside of valid range", "device", device.Name, "temp", data.TempF)
			w.WriteHeader(http.StatusOK)
			return
		}

		db.AddTempReading(data.MacAddress, data.TempF)

		slog.Info("New reading", "device", device.Name, "temp", data.TempF)
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
