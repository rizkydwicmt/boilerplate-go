package main

import (
	"boilerplate-go/internal/pkg/logger"
	"boilerplate-go/internal/pkg/rabbitmq"
	"boilerplate-go/internal/service/worker"
	"context"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	logger.Setup()
	rb, err := rabbitmq.NewConnectionManager(ctx, &rabbitmq.Config{
		Username: "test",
		Password: "test",
		Host:     "localhost",
		Port:     5672,
	})
	if err != nil {
		panic(err)
	}

	publisher, err := rabbitmq.NewPublisher(ctx, rb)
	if err != nil {
		panic(err)
	}

	s, err := worker.NewService(ctx, rb, publisher)
	if err != nil {
		panic(err)
	}

	i := 0
	go func() {
		for {
			i++
			if err := s.SamplePublishMessageRabbit(i); err != nil {
				logger.Error.Println(err)
				panic(err)
			}

		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	cancel()
}
