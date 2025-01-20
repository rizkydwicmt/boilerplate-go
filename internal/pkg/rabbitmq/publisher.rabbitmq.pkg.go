package rabbitmq

import (
	"boilerplate-go/internal/pkg/logger"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	connManager    *ConnectionManager
	channelManager *ChannelManager
	mu             sync.Mutex
	wg             sync.WaitGroup
	maxRetries     int
	retryInterval  time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
}

type PublishOptions struct {
	QueueOpts    *QueueConfig
	QueueName    string
	Exchange     string
	Mandatory    bool
	Immediate    bool
	MaxRetries   int
	RetryBackoff time.Duration
	BatchSize    int
	IsRPC        bool
}

func DefaultPublishOptions(queueName string) *PublishOptions {
	return &PublishOptions{
		QueueOpts:    nil,
		Exchange:     "",
		QueueName:    queueName,
		Mandatory:    false,
		Immediate:    false,
		MaxRetries:   3,
		RetryBackoff: time.Second * 10,
		BatchSize:    100,
		IsRPC:        false,
	}
}

func NewPublisher(ctx context.Context, connManager *ConnectionManager) (*Publisher, error) {
	ctx, cancel := context.WithCancel(ctx)

	pub := &Publisher{
		connManager:    connManager,
		maxRetries:     3,
		retryInterval:  time.Second * 2,
		ctx:            ctx,
		cancel:         cancel,
		channelManager: NewChannelManager(ctx, connManager),
	}

	return pub, nil
}

func (p *Publisher) declareQueue(name string, isRPC bool, config *QueueConfig) (*amqp.Queue, error) {
	ch, err := p.channelManager.GetChannel()
	queueName := name
	cfg := config

	if err != nil || ch == nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	if cfg == nil {
		cfg = DefaultQueueConfig()
		if isRPC {
			cfg.AutoDelete = true
		}
	}

	if isRPC {
		queueName = ""
	}

	reply, err := ch.QueueDeclare(
		queueName,
		cfg.Durable,
		cfg.AutoDelete,
		cfg.Exclusive,
		cfg.NoWait,
		cfg.Args,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	return &reply, nil
}

func (p *Publisher) Publish(msg *Message, opts *PublishOptions) (interface{}, error) {
	return p.PublishWithContext(p.ctx, msg, opts)
}

func (p *Publisher) PublishWithContext(ctx context.Context, msg *Message, opts *PublishOptions) (interface{}, error) {
	if opts.MaxRetries == 0 {
		opts.MaxRetries = p.maxRetries
	}
	if opts.RetryBackoff == 0 {
		opts.RetryBackoff = p.retryInterval
	}

	var lastErr error
	var replyQueue *amqp.Queue

	for attempt := 0; attempt <= opts.MaxRetries; attempt++ {
		if attempt > 0 {
			if err := p.waitForRetry(ctx, opts, attempt); err != nil {
				return nil, err
			}
		}

		ch, err := p.channelManager.GetChannel()
		if err != nil || ch == nil {
			lastErr = err
			logger.Warning.Printf("Failed to setup channel on attempt %d: %v\n", attempt, err)
			continue
		}

		if opts.QueueName != "" {
			var err error
			replyQueue, err = p.declareQueue(opts.QueueName, opts.IsRPC, opts.QueueOpts)
			if err != nil {
				lastErr = err
				logger.Warning.Printf("Failed to declare queue on attempt %d: %v\n", attempt, err)
				continue
			}
		}

		if opts.IsRPC {
			if replyQueue == nil {
				return nil, errors.New("reply queue is not initialized")
			}
			payload := msg.GenerateRPCPayload(opts.QueueName, replyQueue.Name)
			if err := p.publishMessage(ctx, opts, payload); err != nil {
				lastErr = err
				logger.Warning.Printf("Failed to publish message on attempt %d: %v\n", attempt, err)
				continue
			}
			return p.consumeReplyQueue(replyQueue, payload)
		} else {
			payload := msg.GeneratePayload()
			if err := p.publishMessage(ctx, opts, payload); err != nil {
				lastErr = err
				logger.Warning.Printf("Failed to publish message on attempt %d: %v\n", attempt, err)
				continue
			}
		}

		return nil, nil
	}

	return nil, fmt.Errorf("failed to publish message after %d attempts: %w", opts.MaxRetries, lastErr)
}

func (p *Publisher) waitForRetry(ctx context.Context, opts *PublishOptions, attempt int) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("context canceled during retry: %w", ctx.Err())
	case <-time.After(opts.RetryBackoff * time.Duration(attempt)):
	}
	return nil
}

func (p *Publisher) publishMessage(ctx context.Context, opts *PublishOptions, payload *amqp.Publishing) error {
	ch, err := p.channelManager.GetChannel()

	if err != nil || ch == nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}

	err = ch.PublishWithContext(
		ctx,
		opts.Exchange,
		opts.QueueName,
		opts.Mandatory,
		opts.Immediate,
		*payload,
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func (p *Publisher) consumeReplyQueue(replyQueue *amqp.Queue, payload *amqp.Publishing) (interface{}, error) {
	ch, err := p.channelManager.GetChannel()

	if err != nil || ch == nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	msgs, err := ch.Consume(
		replyQueue.Name,
		"",
		true,
		true,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to consume reply queue: %w", err)
	}

	for m := range msgs {
		if m.CorrelationId == payload.CorrelationId {
			var response map[string]interface{}
			if err := json.Unmarshal(m.Body, &response); err != nil {
				return nil, fmt.Errorf("failed to unmarshal response: %w", err)
			}
			return response, nil
		}
	}

	return nil, errors.New("no matching response in reply queue")
}

func (p *Publisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.channelManager.Close(); err != nil {
		return fmt.Errorf("failed to close channel: %w", err)
	}

	p.cancel()

	return nil
}
