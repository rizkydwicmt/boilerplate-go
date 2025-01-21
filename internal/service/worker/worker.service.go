package worker

import (
	"boilerplate-go/internal/pkg/logger"
	"boilerplate-go/internal/pkg/rabbitmq"
	"context"
)

type Service struct {
	ctx       context.Context
	rabbitmq  *rabbitmq.ConnectionManager
	publisher *rabbitmq.Publisher
}

type IService interface {
	SamplePublishMessageRabbit(i int) error
	SampleSubscribeMessageRabbit() error

	SampleMessageSenderRPCRabbit() error
	SampleMessageReceiverRPCRabbit() error
}

func NewService(ctx context.Context, manager *rabbitmq.ConnectionManager, publisher *rabbitmq.Publisher) (IService, error) {
	return &Service{
		ctx:       ctx,
		rabbitmq:  manager,
		publisher: publisher,
	}, nil
}

func (s *Service) SamplePublishMessageRabbit(i int) error {
	opts := rabbitmq.DefaultPublishOptions("queueName")

	msg, err := rabbitmq.NewMessage(map[string]interface{}{
		"key": "value",
		"i":   i,
	}, nil)

	if err != nil {
		return err
	}

	_, err = s.publisher.PublishWithContext(s.ctx, msg, opts)
	// err = s.publisher.SamplePublish(msg, opts)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) SampleSubscribeMessageRabbit() error {
	opts := rabbitmq.DefaultSubscribeOptions("queueName", false)
	subscriber, err := rabbitmq.NewSubscriber(s.ctx, s.rabbitmq, sampleSubscribeMessageRabbitHandler, opts)
	if err != nil {
		return err
	}

	if err := subscriber.Start(); err != nil {
		err = subscriber.Stop()
		if err != nil {
			logger.Error.Println("Failed to stop subscriber: ", err)
			return err
		}
		return err
	}

	return nil
}

func (s *Service) SampleMessageSenderRPCRabbit() error {
	opts := rabbitmq.DefaultPublishOptions("queueNameRPC")
	opts.IsRPC = true
	msg, err := rabbitmq.NewMessage(map[string]interface{}{
		"key": "value",
	}, nil)

	if err != nil {
		return err
	}

	response, err := s.publisher.PublishWithContext(s.ctx, msg, opts)
	if err != nil {
		return err
	}

	logger.Debug.Println("Response: ", response)

	return nil
}

func (s *Service) SampleMessageReceiverRPCRabbit() error {
	opts := rabbitmq.DefaultSubscribeOptions("queueNameRPC", true)
	subscriber, err := rabbitmq.NewSubscriber(s.ctx, s.rabbitmq, sampleMessageRabbitRPCHandler, opts)
	if err != nil {
		return err
	}

	if err := subscriber.Start(); err != nil {
		err = subscriber.Stop()
		if err != nil {
			logger.Error.Println("Failed to stop subscriber: ", err)
			return err
		}
		return err
	}

	return nil
}
