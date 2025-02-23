package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")
	connectionString := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connectionString)
	if err != nil {
		log.Fatal(fmt.Errorf("could not connect to %s", connectionString))
	}
	defer conn.Close()
	fmt.Printf("connection to RabbitMQ was successful\n")

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	fmt.Printf("\nShutting down connection\n")
}
