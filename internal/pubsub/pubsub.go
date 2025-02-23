package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	SimpleQueueDurable = iota
	SimpleQueueTransient
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	jsonBytes, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("could not encode data")
	}
	ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        jsonBytes,
	})
	return nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
	handler func(T),
) error {
	channel, _, err := DeclareAndBind(conn, exchange, queueName, key, int(simpleQueueType))
	if err != nil {
		return fmt.Errorf("could not bind queueName: %s", err)
	}
	ch, err := channel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("could not consume channel: %s", err)
	}
	go func() {
		defer channel.Close()
		for msg := range ch {
			var target T
			err = json.Unmarshal(msg.Body, &target)
			if err != nil {
				fmt.Printf("could not JSON decode message: %s", err)
			}
			handler(target)
			msg.Ack(false)
		}
	}()
	return nil
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not create channel in queue declaration")
	}
	var durable, autoDelete, exclusive, noWait bool
	if simpleQueueType == SimpleQueueDurable {
		durable = true
		autoDelete = false
		exclusive = false
	} else {
		durable = false
		autoDelete = true
		exclusive = true
	}
	noWait = false

	queue, err := channel.QueueDeclare(queueName, durable, autoDelete, exclusive, noWait, nil)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not declare queue")
	}
	err = channel.QueueBind(queueName, key, exchange, noWait, nil)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not bind queue")
	}
	return channel, queue, nil
}
