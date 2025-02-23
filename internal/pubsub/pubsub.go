package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ActType int
type SimpleQueueType int

const (
	SimpleQueueDurable = iota
	SimpleQueueTransient
)

const (
	Ack = iota
	NackRequeue
	NackDiscard
)

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var buffer bytes.Buffer

	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(val)
	if err != nil {
		return fmt.Errorf("could not gob encode data: %v", err)
	}
	ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "application/gob",
		Body:        buffer.Bytes(),
	})
	return nil
}

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	jsonBytes, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("could not json encode data: %v", err)
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
	handler func(T) ActType,
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
			ackType := handler(target)
			switch ackType {
			case Ack:
				fmt.Print("Sending Ack\n")
				msg.Ack(false)
			case NackDiscard:
				fmt.Print("Sending NackDiscard\n")
				msg.Nack(false, false)
			case NackRequeue:
				fmt.Print("Sending NackRequeue\n")
				msg.Nack(false, true)
			}
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
	queue, err := channel.QueueDeclare(queueName, durable, autoDelete, exclusive, noWait, amqp.Table{"x-dead-letter-exchange": "peril_dlx"})
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not declare queue")
	}
	err = channel.QueueBind(queueName, key, exchange, noWait, nil)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not bind queue")
	}
	return channel, queue, nil
}
