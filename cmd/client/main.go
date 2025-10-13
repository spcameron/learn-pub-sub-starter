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

	const rabbitConnectionString = "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(rabbitConnectionString)
	if err != nil {
		log.Fatalf("amqp/Dial error: %v", err)
	}
	defer conn.Close()
	fmt.Println("Connection successful...")

	publishCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("amqp/Channel error: %v", err)
	}

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("gamelogic/ClientWelcome error: %v", err)
	}
	gamestate := gamelogic.NewGameState(username)

	if err := pubsub.SubscribeJSON(
		conn,
		string(routing.ExchangePerilDirect),
		"pause."+username,
		string(routing.PauseKey),
		pubsub.SimpleQueueTransient,
		handlerPause(gamestate),
	); err != nil {
		log.Fatalf("pubsub/SubscribeJSON error: %v", err)
	}

	if err := pubsub.SubscribeJSON(
		conn,
		string(routing.ExchangePerilTopic),
		"army_moves."+username,
		"army_moves.*",
		pubsub.SimpleQueueTransient,
		handlerMove(gamestate),
	); err != nil {
		log.Fatalf("pubsub/SubscribeJSON error: %v", err)
	}

	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		cmd := words[0]
		switch cmd {
		case "spawn":
			if err := gamestate.CommandSpawn(words); err != nil {
				fmt.Printf("gamestate/CommandSpawn error: %v\n", err)
				continue
			}
		case "move":
			mv, err := gamestate.CommandMove(words)
			if err != nil {
				fmt.Printf("gamestate/CommandMove error: %v\n", err)
				continue
			}

			if err := pubsub.PublishJSON(
				publishCh,
				string(routing.ExchangePerilTopic),
				"army_moves."+username,
				mv,
			); err != nil {
				fmt.Printf("pubsub/PublishJSON error: %v\n", err)
				continue
			}

			fmt.Print("Move published successfully.")
		case "status":
			gamestate.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Printf("Unrecognized command: %s", cmd)
			continue
		}
	}
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
	}
}

func handlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) {
	return func(mv gamelogic.ArmyMove) {
		defer fmt.Print("> ")
		gs.HandleMove(mv)
	}
}
