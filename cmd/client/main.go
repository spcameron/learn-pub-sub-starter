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
		handlerMove(gamestate, publishCh),
	); err != nil {
		log.Fatalf("pubsub/SubscribeJSON error: %v", err)
	}

	if err := pubsub.SubscribeJSON(
		conn,
		string(routing.ExchangePerilTopic),
		"war",
		string(routing.WarRecognitionsPrefix)+".*",
		pubsub.SimpleQueueDurable,
		handlerWarMessages(gamestate),
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

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.Acktype {
	return func(ps routing.PlayingState) pubsub.Acktype {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.ArmyMove) pubsub.Acktype {
	return func(mv gamelogic.ArmyMove) pubsub.Acktype {
		defer fmt.Print("> ")
		mvOutcome := gs.HandleMove(mv)
		switch mvOutcome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			if err := pubsub.PublishJSON(
				ch,
				routing.ExchangePerilTopic,
				string(routing.WarRecognitionsPrefix)+"."+mv.Player.Username,
				gamelogic.RecognitionOfWar{
					Attacker: mv.Player,
					Defender: gs.GetPlayerSnap(),
				},
			); err != nil {
				log.Printf("pubsub/PublishJSON error: %v", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		default:
			return pubsub.NackDiscard
		}
	}
}

func handlerWarMessages(gs *gamelogic.GameState) func(gamelogic.RecognitionOfWar) pubsub.Acktype {
	return func(rw gamelogic.RecognitionOfWar) pubsub.Acktype {
		defer fmt.Print("> ")
		outcome, _, _ := gs.HandleWar(rw)
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			return pubsub.Ack
		case gamelogic.WarOutcomeYouWon:
			return pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			return pubsub.Ack
		default:
			fmt.Printf("unrecognized outcome: %+v", outcome)
			return pubsub.NackDiscard
		}
	}
}
