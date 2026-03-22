package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
)

type TelemetryData struct {
	DeviceID      string  `json:"device_id"`
	Timestamp     string  `json:"timestamp"`
	SensorType    string  `json:"sensor_type"`
	ReadingNature string  `json:"reading_nature"` // "discrete" or "analog"
	Value         float64 `json:"value"`
}

var (
	rabbitConn *amqp.Connection
	rabbitCh   *amqp.Channel

	// Variables for easier testing
	chPool     chan *amqp.Channel
	publishMsg = func(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
		ch := <-chPool
		err := ch.Publish(exchange, key, mandatory, immediate, msg)
		chPool <- ch
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
		log.Printf("Failed to connect to RabbitMQ, retrying in 5s... %v", err)
		time.Sleep(5 * time.Second)
	}

	if rabbitConn == nil {
		log.Fatal("Could not connect to RabbitMQ after retries")
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

	// Create a pool of channels to avoid lock contention under high RPS
	poolSize := 100
	chPool = make(chan *amqp.Channel, poolSize)
	for i := 0; i < poolSize; i++ {
		ch, err := rabbitConn.Channel()
		if err != nil {
			log.Fatal(err)
		}
		chPool <- ch
	}
}

func ingestHandler(c *gin.Context) {
	// Read payload directly. Saves the cost of Unmarshalling and Marshaling just to proxy it.
	body, err := io.ReadAll(c.Request.Body)
	if err != nil || len(body) == 0 {
		c.JSON(400, gin.H{"error": "Bad Request"})
		return
	}

	// Validate JSON fast without reflection
	if !json.Valid(body) {
		c.JSON(400, gin.H{"error": "Bad Request"})
		return
	}

	err = publishMsg(
		"",                // exchange
		"telemetry_queue", // routing key
		false,             // mandatory
		false,             // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		c.JSON(500, gin.H{"error": "Could not publish message"})
		return
	}

	c.JSON(202, gin.H{"message": "Data accepted for processing"})
}

func main() {
	// Set Gin to release mode to disable debug output, improving performance
	gin.SetMode(gin.ReleaseMode)

	initRabbitMQ()

	defer rabbitConn.Close()
	defer rabbitCh.Close()

	// Use gin.New() instead of gin.Default() to avoid the default Logger middleware,
	// which is a huge bottleneck (writes every request to stdout synchronously).
	r := gin.New()
	r.Use(gin.Recovery())

	r.POST("/ingest", ingestHandler)

	log.Println("Server starts listening on port 8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
