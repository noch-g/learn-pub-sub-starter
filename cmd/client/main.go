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
	fmt.Println("Starting Peril client...")
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

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatal(fmt.Errorf("could not get username from client welcome"))
	}

	gs := gamelogic.NewGameState(username)
	pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, routing.PauseKey+"."+username, routing.PauseKey, pubsub.SimpleQueueTransient, handlerPause(gs))
	pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, routing.ArmyMovesPrefix+"."+username, routing.ArmyMovesPrefix+".*", pubsub.SimpleQueueTransient, handlerMove(gs))

forloop:
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "spawn":
			err = gs.CommandSpawn(words)
			if err != nil {
				fmt.Println(err)
				continue
			}
		case "move":
			move, err := gs.CommandMove(words)
			if err != nil {
				fmt.Println(err)
				continue
			}
			err = pubsub.PublishJSON(channel, routing.ExchangePerilTopic, routing.ArmyMovesPrefix+"."+username, move)
			if err != nil {
				fmt.Println("move was successfully published")
			}
		case "status":
			gs.CommandStatus()
		case "spam":
			fmt.Printf("Spamming is not allowed yet")
		case "help":
			gamelogic.PrintClientHelp()
		case "quit":
			gamelogic.PrintQuit()
			break forloop
		default:
			fmt.Print("Unknown command")
		}
	}

	fmt.Printf("Shutting down connection\n")
}
