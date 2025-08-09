#include <ESP8266WiFi.h>
#include <ESP8266HTTPClient.h>
#include <OneWire.h>
#include <DallasTemperature.h>

#include "secrets.h"

// HTTPS endpoint
const char* endpoint = "https://weather.casanv2.net/temperature";
const char fingerprint[] = "F0 7F 10 12 D9 36 7A 19 A9 CC 13 65 35 BA 59 A1 8B 81 B7 D4";

// DS18B20 on GPIO5 (D1)
const int ONE_WIRE_BUS = 5;
OneWire oneWire(ONE_WIRE_BUS);
DallasTemperature sensors(&oneWire);

void setup() {
  // Serial.begin(115200);
  delay(500);
  // Serial.println();

  WiFi.mode(WIFI_STA);
  WiFi.begin(WIFI_SSID, WIFI_PASS);
  int retries = 0;
  while (WiFi.status() != WL_CONNECTED && retries < 20) {
    delay(500);
    // Serial.print(".");
    retries++;
  }

  if (WiFi.status() == WL_CONNECTED) {
    HTTPClient http;
    // WiFiClient client; // http

    WiFiClientSecure client; // https
    client.setFingerprint(fingerprint);

    http.begin(client, endpoint);
    http.addHeader("Content-Type", "application/json");

    // Start temperature sensor
    sensors.begin();
    sensors.requestTemperatures();
    float tempf = sensors.getTempFByIndex(0);

    String mac = WiFi.macAddress();
    String payload = "{\"mac\": \"" + mac + "\", \"temperature\": " + tempf + "}";

    int httpResponseCode = http.POST(payload);
    // Serial.print("\nHTTP POST code: ");
    // Serial.println(httpResponseCode);
    http.end();
  } else {
    // Serial.println("WiFi failed");
  }

  delay(100);
  // µs ( 2 minutes)
  ESP.deepSleep(120e6);
  // ESP.deepSleep(10e6);
}

void loop() {}
