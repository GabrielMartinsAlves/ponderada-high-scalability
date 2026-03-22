package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type TelemetryData struct {
	DeviceID      string  `json:"device_id"`
	Timestamp     string  `json:"timestamp"`
	SensorType    string  `json:"sensor_type"`
	ReadingNature string  `json:"reading_nature"` // "discrete" or "analog"
	Value         float64 `json:"value"`
}

// Variables to allow mocking in tests
var (
	rabbitConn *amqp.Connection
	rabbitCh   *amqp.Channel
	db         *sql.DB
	stmt       *sql.Stmt

	insertDBMsg = func(data TelemetryData, parsedTime time.Time) error {
		// Use prepared statement for significantly faster repeated inserts
		_, err := stmt.Exec(
			data.DeviceID, parsedTime, data.SensorType, data.ReadingNature, data.Value,
		)
		return err
	}
)

func initRabbitMQ() {
	var err error
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@localhost:5672/"
	}

	for i := 0; i < 10; i++ {
		rabbitConn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Printf("Consumer - Failed to connect to RabbitMQ, retrying in 5s... %v", err)
		time.Sleep(5 * time.Second)
	}

	if rabbitConn == nil {
		log.Fatal("Consumer - Could not connect to RabbitMQ after retries")
	}

	rabbitCh, err = rabbitConn.Channel()
	if err != nil {
		log.Fatal(err)
	}

	_, err = rabbitCh.QueueDeclare(
		"telemetry_queue", // name
		true,              // durable
		false,             // delete when unused
		false,             // exclusive
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		log.Fatal(err)
	}
}

func initDB() {
	var err error
	url := os.Getenv("DB_URL")
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/telemetry?sslmode=disable"
	}

	db, err = sql.Open("postgres", url)
	if err != nil {
		log.Fatal(err)
	}

	// Optimize connection pool for high concurrency
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(50)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Ping database indefinitely
	for {
		err = db.Ping()
		if err == nil {
			break
		}
		log.Println("Consumer - Database not ready, retrying...")
		time.Sleep(2 * time.Second)
	}

	// Prepare the statement once to save Postgres parsing overhead on every insert
	stmt, err = db.Prepare(`INSERT INTO telemetry_data (device_id, recorded_at, sensor_type, reading_nature, value) VALUES ($1, $2, $3, $4, $5)`)
	if err != nil {
		log.Fatal(err)
	}
}

func processMessage(body []byte) (bool, bool) {
	var data TelemetryData
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("Consumer - Error unmarshalling data: %s", err)
		return false, false // Nack without requeue
	}

	parsedTime, err := time.Parse(time.RFC3339, data.Timestamp)
	if err != nil {
		parsedTime = time.Now()
	}

	if err := insertDBMsg(data, parsedTime); err != nil {
		log.Printf("Consumer - Error inserting into DB: %s", err)
		return false, true // Nack and requeue
	}

	// log.Println("Consumer - Data persisted for device", data.DeviceID) // Removed to save IO overhead
	return true, false // Ack
}

func startConsumer() {
	// Set prefetch count to allow RabbitMQ to send messages in batches,
	// preventing the consumer from being starved or overwhelmed.
	err := rabbitCh.Qos(
		1000,  // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		log.Printf("Consumer - Failed to set QoS: %s", err)
	}

	msgs, err := rabbitCh.Consume(
		"telemetry_queue", // queue
		"",                // consumer
		false,             // auto-ack
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // args
	)
	if err != nil {
		log.Fatal(err)
	}

	// Spin up 50 concurrent worker goroutines to process messages
	// This drastically increases throughput compared to a single sequential loop.
	workerCount := 50
	for i := 0; i < workerCount; i++ {
		go func() {
			for d := range msgs {
				ack, requeue := processMessage(d.Body)
				if ack {
					d.Ack(false)
				} else {
					d.Nack(false, requeue)
				}
			}
		}()
	}

	// Keep the main goroutine alive
	select {}
}

func main() {
	initDB()
	initRabbitMQ()

	defer db.Close()
	defer rabbitConn.Close()
	defer rabbitCh.Close()

	log.Println("Consumer started. To exit press CTRL+C")
	startConsumer()
}
