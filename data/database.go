package data

import (
	"database/sql"
	"errors"
	"log"
	"log/slog"
	"sync"
	"time"
)

type Device struct {
	ID   string
	Name string
}

type TemperatureReading struct {
	TempF     float32
	Timestamp time.Time
}

type InMemoryCache struct {
	mu      sync.Mutex
	devices map[string]Device
}

type Database struct {
	db    *sql.DB
	cache *InMemoryCache
}

var (
	dbInstance *Database
	once       sync.Once
)

func NewDatabase(db *sql.DB) *Database {
	once.Do(func() {
		dbInstance = &Database{
			db: db,
			cache: &InMemoryCache{
				devices: make(map[string]Device),
			},
		}

		dbInstance.runMigrations()
	})

	return dbInstance
}

func (d *Database) runMigrations() {
	var statements = []string{
		"CREATE TABLE IF NOT EXISTS devices(id TEXT NOT NULL PRIMARY KEY, name TEXT)",
		"CREATE TABLE IF NOT EXISTS temperature_readings(id INTEGER PRIMARY KEY, device_id TEXT NOT NULL, temp_f FLOAT, timestamp DATETIME)",
		"CREATE INDEX IF NOT EXISTS idx_temperature_readings_device_id ON temperature_readings(device_id)",
	}

	slog.Info("Running db migrations...")
	for _, stmt := range statements {
		_, err := d.db.Exec(stmt)
		if err != nil {
			log.Fatalf("%q: %s\n", err, stmt)
		}
		slog.Info("Ran migration", "stmt", stmt)
	}
	slog.Info("Database migrations complete!\n\n")
}

func (d *Database) GetDevice(id string) Device {
	device, ok := d.cache.devices[id]
	if ok {
		return device
	}

	row := d.db.QueryRow("select id, name from devices where id = ?", id)

	if err := row.Scan(&device.ID, &device.Name); err != nil {
		if errors.Is(sql.ErrNoRows, err) {
			device = Device{ID: id, Name: id}
		} else {
			log.Fatal(err)
		}
	}

	d.cache.mu.Lock()
	defer d.cache.mu.Unlock()
	d.cache.devices[id] = device

	return device
}

func (d *Database) AddTempReading(deviceID string, tempF float32) error {
	now := time.Now()

	_, err := d.db.Exec("INSERT INTO temperature_readings (device_id, temp_f, timestamp) VALUES (?, ?, ?)", deviceID, tempF, now)
	if err != nil {
		return err
	}

	return nil
}

func (d *Database) GetTemperatureReadings() map[string][]TemperatureReading {
	rows, err := d.db.Query("select devices.name, r.temp_f, r.timestamp from devices JOIN temperature_readings AS r ON devices.id = r.device_id order by r.timestamp desc")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var readings = make(map[string][]TemperatureReading)

	for rows.Next() {
		var reading TemperatureReading
		var deviceName string
		if err := rows.Scan(&deviceName, &reading.TempF, &reading.Timestamp); err != nil {
			log.Fatal(err)
		}

		readings[deviceName] = append(readings[deviceName], reading)
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return readings
}
