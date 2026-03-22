package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestIngestHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		payload        interface{}
		mockPublishErr error
		expectedStatus int
	}{
		{
			name: "Valid Payload",
			payload: TelemetryData{
				DeviceID:      "device_1",
				Timestamp:     "2023-01-01T12:00:00Z",
				SensorType:    "temperature",
				ReadingNature: "analog",
				Value:         25.5,
			},
			mockPublishErr: nil,
			expectedStatus: 202,
		},
		{
			name:           "Invalid JSON",
			payload:        "not-a-json",
			mockPublishErr: nil,
			expectedStatus: 400,
		},
		{
			name: "Publish Error",
			payload: TelemetryData{
				DeviceID:      "device_1",
				Timestamp:     "2023-01-01T12:00:00Z",
				SensorType:    "temperature",
				ReadingNature: "analog",
				Value:         25.5,
			},
			mockPublishErr: errors.New("rabbit is down"),
			expectedStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the publishMsg function
			publishMsg = func(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
				return tt.mockPublishErr
			}

			r := gin.New()
			r.POST("/ingest", ingestHandler)

			var reqBody []byte
			if strPayload, ok := tt.payload.(string); ok {
				reqBody = []byte(strPayload)
			} else {
				reqBody, _ = json.Marshal(tt.payload)
			}

			req, _ := http.NewRequest(http.MethodPost, "/ingest", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
