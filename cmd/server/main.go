package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
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

	channel, err := conn.Channel()
	if err != nil {
		log.Fatal(fmt.Errorf("channel could not be created"))
	}

	_, queue, err := pubsub.DeclareAndBind(conn, routing.ExchangePerilTopic, routing.GameLogSlug, routing.GameLogSlug+".*", pubsub.SimpleQueueDurable)
	if err != nil {
		log.Fatal(fmt.Errorf("could not subscribe to pause: %s", err))
	}
	fmt.Printf("Queue %v declared and bound!\n", queue.Name)

	gamelogic.PrintServerHelp()
forloop:
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "pause":
			fmt.Print("Publish pause message\n")
			err = pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true})
			if err != nil {
				log.Fatal(fmt.Errorf("error in PublishJSON"))
			}
		case "resume":
			fmt.Print("Publish resume message\n")
			err = pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: false})
			if err != nil {
				log.Fatal(fmt.Errorf("error in PublishJSON"))
			}
		case "quit":
			fmt.Print("Quitting server\n")
			break forloop
		default:
			fmt.Print("Unknown command")
		}
	}
	fmt.Printf("Shutting down connection\n")
}
