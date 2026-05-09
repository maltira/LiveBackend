package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"user/config"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	conn *amqp.Connection
	ch   *amqp.Channel
)

func InitRabbitMQ() {
	addr := config.Env.RabbitMQAddr
	var err error
	conn, err = amqp.Dial(addr)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to RabbitMQ: %v", err))
	}

	ch, err = conn.Channel()
	if err != nil {
		panic(fmt.Sprintf("Failed to open channel: %v", err))
	}
}

func Publish(event string, payload []byte) error {
	q, err := ch.QueueDeclare(
		event, // имя очереди
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	return ch.PublishWithContext(
		context.Background(),
		"",     // exchange
		q.Name, // routing key
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        payload,
		},
	)
}

func Consume(event string, handler func([]byte)) error {
	q, err := ch.QueueDeclare(event, true, false, false, false, nil)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			handler(msg.Body)
			err = msg.Ack(false)
		}
	}()

	return nil
}

func Close() {
	if ch != nil {
		_ = ch.Close()
	}
	if conn != nil {
		_ = conn.Close()
	}
	log.Println("RabbitMQ connection closed")
}
