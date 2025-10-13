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
	fmt.Println("Starting Peril server...")

	connectionString := "amqp://guest:guest@localhost:5672/"
	connection, err := amqp.Dial(connectionString)
	if err != nil {
		log.Fatalf("amqp/Dial error: %v", err)
	}
	defer connection.Close()

	channel, err := connection.Channel()
	if err != nil {
		log.Fatalf("amqp/Channel error: %v", err)
	}

	fmt.Println("Connection successful...")
	fmt.Println("Declaring queue...")

	queue, err := channel.QueueDeclare(
		routing.GameLogSlug,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("amqp/QueueDeclare error: %v", err)
	}

	if err := channel.QueueBind(
		queue.Name,
		routing.GameLogSlug+".*",
		routing.ExchangePerilTopic,
		false,
		nil,
	); err != nil {
		log.Fatalf("amqp/QueueBind error: %v", err)
	}

	gamelogic.PrintServerHelp()

	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		cmd := words[0]
		switch cmd {
		case "pause":
			fmt.Println("Sending pause message...")
			commandPause(channel)
		case "resume":
			fmt.Println("Sending resume message...")
			commandResume(channel)
		case "quit":
			fmt.Println("Exiting program...")
			os.Exit(0)
		default:
			fmt.Printf("Unrecognized command: %s", cmd)
			continue
		}
	}
}

func commandPause(channel *amqp.Channel) {
	if err := pubsub.PublishJSON(
		channel,
		routing.ExchangePerilDirect,
		routing.PauseKey,
		routing.PlayingState{
			IsPaused: true,
		},
	); err != nil {
		log.Fatalf("pubsub/PublishJSON error: %v", err)
	}
}

func commandResume(channel *amqp.Channel) {
	if err := pubsub.PublishJSON(
		channel,
		routing.ExchangePerilDirect,
		routing.PauseKey,
		routing.PlayingState{
			IsPaused: false,
		},
	); err != nil {
		log.Fatalf("pubsub/PublishJSON error: %v", err)
	}
}
