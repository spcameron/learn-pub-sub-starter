package main

import (
	"fmt"
	"log"
	"os"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")

	connectionString := "amqp://guest:guest@localhost:5672/"
	connection, err := amqp.Dial(connectionString)
	if err != nil {
		log.Fatalf("amqp/Dial error: %v", err)
	}
	defer connection.Close()

	fmt.Println("Connection successful...")

	name, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("gamelogic/ClientWelcome error: %v", err)
	}

	channel, queue, err := pubsub.DeclareAndBind(
		connection,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+name,
		routing.PauseKey,
		pubsub.SimpleQueueTransient,
	)
	if err != nil {
		log.Fatalf("pubsub/DeclareAndBind error: %v", err)
	}
	defer channel.Close()

	log.Printf("queue name: %s", queue.Name)

	gamestate := gamelogic.NewGameState(name)

	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		cmd := words[0]
		switch cmd {
		case "spawn":
			if err := gamestate.CommandSpawn(words); err != nil {
				log.Fatalf("gamestate/CommandSpawn error: %v", err)
			}
		case "move":
			mv, err := gamestate.CommandMove(words)
			if err != nil {
				log.Fatalf("gamestate/CommandMove error: %v", err)
			}
			fmt.Printf("Move successful: %v", mv)
		case "status":
			gamestate.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			os.Exit(0)
		default:
			fmt.Printf("Unrecognized command: %s", cmd)
			continue
		}
	}
}
