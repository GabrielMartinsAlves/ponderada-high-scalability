package main

import (
	"errors"
	"testing"
	"time"
)

func TestProcessMessage(t *testing.T) {
	tests := []struct {
		name            string
		body            []byte
		mockInsertErr   error
		expectedAck     bool
		expectedRequeue bool
	}{
		{
			name:            "Valid Data Success",
			body:            []byte(`{"device_id":"sensor_1","timestamp":"2023-10-31T14:20:00Z","sensor_type":"temperature","reading_nature":"analog","value":25.5}`),
			mockInsertErr:   nil,
			expectedAck:     true,
			expectedRequeue: false,
		},
		{
			name:            "Invalid JSON Structure",
			body:            []byte(`{"device_id":"sensor_1", invalid...`),
			mockInsertErr:   nil,
			expectedAck:     false,
			expectedRequeue: false,
		},
		{
			name:            "Database Insert Failure",
			body:            []byte(`{"device_id":"sensor_2","timestamp":"2023-10-31T14:20:00Z","sensor_type":"presence","reading_nature":"discrete","value":1}`),
			mockInsertErr:   errors.New("db connection lost"),
			expectedAck:     false,
			expectedRequeue: true,
		},
		{
			name:            "Invalid Timestamp Parsing (Fallbacks to Now)",
			body:            []byte(`{"device_id":"sensor_3","timestamp":"wrong-format-time","sensor_type":"presence","reading_nature":"discrete","value":1}`),
			mockInsertErr:   nil,
			expectedAck:     true,
			expectedRequeue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock DB layer
			insertDBMsg = func(data TelemetryData, parsedTime time.Time) error {
				return tt.mockInsertErr
			}

			ack, requeue := processMessage(tt.body)

			if ack != tt.expectedAck {
				t.Errorf("Expected ack %v, got %v", tt.expectedAck, ack)
			}

			if requeue != tt.expectedRequeue {
				t.Errorf("Expected requeue %v, got %v", tt.expectedRequeue, requeue)
			}
		})
	}
}
