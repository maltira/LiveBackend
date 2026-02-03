package rabbitmq

import (
	"auth/config"
	"context"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	conn *amqp.Connection
	ch   *amqp.Channel
)

func InitRabbitMQ() {
	addr := config.Env.RabbitMQAddr
	conn, err := amqp.Dial(addr)
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

func Close() {
	if ch != nil {
		_ = ch.Close()
	}
	if conn != nil {
		_ = conn.Close()
	}
	log.Println("RabbitMQ connection closed")
}
