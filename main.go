package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"time"

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

type TempTableData struct {
	Headers []string
	Rows    [][]*data.TemperatureReading
}

var db *data.Database

func FormatDate(t time.Time) string {
	return t.Format("2006-01-02 03:04:05 PM")
}

func makeTempTable(tempReadings map[string][]data.TemperatureReading) TempTableData {
	headers := make([]string, 0, len(tempReadings))
	for h := range tempReadings {
		headers = append(headers, h)
	}
	slices.Sort(headers)

	maxLen := 0
	for _, readings := range tempReadings {
		if len(readings) > maxLen {
			maxLen = len(readings)
		}
	}

	rows := make([][]*data.TemperatureReading, maxLen)
	for i := range maxLen {
		row := make([]*data.TemperatureReading, len(headers))
		for index, header := range headers {
			if i < len(tempReadings[header]) {
				row[index] = &tempReadings[header][i]
			}
		}
		rows[i] = row
	}

	return TempTableData{Headers: headers, Rows: rows}
}

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
		readings := db.GetTemperatureReadings()
		funcs := template.FuncMap{
			"FormatDate": FormatDate,
		}

		tmpl := template.Must(
			template.New("index.html").Funcs(funcs).ParseFiles("html/index.html"),
		)

		tmpl.Execute(w, makeTempTable(readings))
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
