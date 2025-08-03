package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

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
	MacAddress string
	Name       string
}

type TempSensorReading struct {
	Temperature float32 `json:"temperature"`
	Timestamp   string  `json:"timestamp"`
}

var devices []Device = []Device{
	{MacAddress: "2C:F4:32:1A:87:C0", Name: "Office"},
	{MacAddress: "C8:C9:A3:5E:03:8C", Name: "Garage"},
}

var currentReadings map[string]TempSensorReading

func getDeviceName(macAddress string) string {
	for _, device := range devices {
		if device.MacAddress == macAddress {
			return device.Name
		}
	}
	return macAddress
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

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

		deviceName := getDeviceName(data.MacAddress)

		// sometimes the temp can be outside the valid range
		// we will log an error and ignore it as a bad read
		if data.Temperature < minTempF || data.Temperature > maxTempF {
			slog.Error("temperature outside of valid range", "device", deviceName, "temp", data.Temperature)
			w.WriteHeader(http.StatusOK)
			return
		}

		now := time.Now().In(location)
		currentReadings[deviceName] = TempSensorReading{Temperature: data.Temperature, Timestamp: now.String()}

		slog.Info("New reading", "device", deviceName, "temp", data.Temperature)
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
