#!/bin/sh
chown -R appuser:appgroup /app/data
exec /app/weather-station
