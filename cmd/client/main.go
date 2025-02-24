package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/noch-g/learn-pub-sub-starter/internal/gamelogic"
	"github.com/noch-g/learn-pub-sub-starter/internal/pubsub"
	"github.com/noch-g/learn-pub-sub-starter/internal/routing"
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

	publishCh, err := conn.Channel()
	if err != nil {
		log.Fatal(fmt.Errorf("channel could not be created"))
	}

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatal(fmt.Errorf("could not get username from client welcome"))
	}

	gs := gamelogic.NewGameState(username)
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix+"."+gs.GetUsername(),
		routing.ArmyMovesPrefix+".*",
		pubsub.SimpleQueueTransient,
		handlerMove(gs, publishCh),
	)
	if err != nil {
		fmt.Printf("could not subscribe to army moves: %v", err)
	}
	pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix,
		routing.WarRecognitionsPrefix+".*",
		pubsub.SimpleQueueDurable,
		handlerWar(gs, publishCh),
	)
	if err != nil {
		fmt.Printf("could not subscribe to war declarations: %v", err)
	}
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+gs.GetUsername(),
		routing.PauseKey,
		pubsub.SimpleQueueTransient,
		handlerPause(gs),
	)
	if err != nil {
		fmt.Printf("could not subscribe to pause: %v", err)
	}

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

			err = pubsub.PublishJSON(
				publishCh,
				routing.ExchangePerilTopic,
				routing.ArmyMovesPrefix+"."+move.Player.Username,
				move,
			)
			if err != nil {
				fmt.Printf("error: %s\n", err)
				continue
			}
			fmt.Printf("Moved %v units to %s\n", len(move.Units), move.ToLocation)
		case "status":
			gs.CommandStatus()
		case "spam":
			if len(words) != 2 {
				fmt.Print("usage: spam <int>\n")
				continue
			}
			nbSpam, err := strconv.Atoi(words[1])
			if err != nil {
				fmt.Printf("could not convert %s to int\n", words[1])
				continue
			}
			var logMessage string
			for range nbSpam {
				logMessage = gamelogic.GetMaliciousLog()
				fmt.Println(logMessage)
				pubsub.PublishGob(publishCh, routing.ExchangePerilTopic, routing.GameLogSlug+"."+gs.GetUsername(), routing.GameLog{
					CurrentTime: time.Now(),
					Message:     logMessage,
					Username:    gs.GetUsername(),
				})
			}
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
