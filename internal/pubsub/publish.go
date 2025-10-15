package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	jsonData, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("json/Marshal error: %w", err)
	}

	if err := ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        jsonData,
		},
	); err != nil {
		return fmt.Errorf("amqp/PublishWithContext error: %w", err)
	}

	return nil
}

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(val); err != nil {
		return fmt.Errorf("gob/Encode error: %w", err)
	}

	if err := ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/gob",
			Body:        buf.Bytes(),
		},
	); err != nil {
		return fmt.Errorf("amqp/PublishWithContext error: %w", err)
	}

	return nil
}

func PublishGameLog(ch *amqp.Channel, username, logMsg string) error {
	log := routing.GameLog{
		CurrentTime: time.Now().UTC(),
		Message:     logMsg,
		Username:    username,
	}

	return PublishGob(
		ch,
		routing.ExchangePerilTopic,
		string(routing.GameLogSlug)+"."+username,
		log,
	)
}
