package worker

import (
	"boilerplate-go/internal/pkg/logger"

	amqp "github.com/rabbitmq/amqp091-go"
)

func sampleSubscribeMessageRabbitHandler(msg *amqp.Delivery) (interface{}, error) {
	logger.Debug.Println("Received message:", string(msg.Body))
	return nil, nil
}

func sampleMessageRabbitRPCHandler(msg *amqp.Delivery) (interface{}, error) {
	logger.Debug.Println("Received message:", string(msg.Body))
	response := map[string]interface{}{
		"key1": "value1",
	}
	return response, nil
}
