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
	conn, err := amqp.Dial(connectionString)
	if err != nil {
		log.Fatalf("amqp/Dial error: %v", err)
	}
	defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("amqp/Channel error: %v", err)
	}

	fmt.Println("Connection successful...")
	fmt.Println("Declaring queue...")

	if err := pubsub.SubscribeGob(
		conn,
		string(routing.ExchangePerilTopic),
		string(routing.GameLogSlug),
		string(routing.GameLogSlug)+".*",
		pubsub.SimpleQueueDurable,
		handlerLogs(),
	); err != nil {
		log.Fatalf("pubsub/SubscribeGob error: %v", err)
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

func handlerLogs() func(routing.GameLog) pubsub.Acktype {
	return func(gamelog routing.GameLog) pubsub.Acktype {
		defer fmt.Print("> ")
		if err := gamelogic.WriteLog(gamelog); err != nil {
			return pubsub.NackRequeue
		}
		return pubsub.Ack
	}
}
