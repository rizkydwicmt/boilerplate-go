package rabbitmq

import (
	_type "boilerplate-go/internal/common/type"
	"boilerplate-go/internal/pkg/helper"
	"boilerplate-go/internal/pkg/logger"
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants/v2"

	amqp "github.com/rabbitmq/amqp091-go"
)

type MessageHandler func(msg *amqp.Delivery) (interface{}, error)

type SubscribeOptions struct {
	QueueOpts     *QueueConfig
	QueueName     string
	ConsumerName  string
	AutoAck       bool
	Exclusive     bool
	NoLocal       bool
	NoWait        bool
	Args          amqp.Table
	WorkerCount   int
	PrefetchCount int
	MessageBuffer int
	IsRPC         bool
}

func DefaultSubscribeOptions(queueName string) *SubscribeOptions {
	return &SubscribeOptions{
		QueueOpts:     nil,
		QueueName:     queueName,
		ConsumerName:  queueName,
		AutoAck:       false,
		Exclusive:     false,
		NoLocal:       false,
		NoWait:        false,
		Args:          nil,
		WorkerCount:   3,
		PrefetchCount: 10,
		MessageBuffer: 100,
		IsRPC:         false,
	}
}

type Subscriber struct {
	connManager     *ConnectionManager
	channelManagers []*ChannelManager
	handler         MessageHandler
	opts            *SubscribeOptions
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	isRunning       atomic.Bool
	pool            *ants.Pool
	mu              sync.RWMutex
	msgChan         chan *amqp.Delivery
}

func NewSubscriber(ctx context.Context, connManager *ConnectionManager, handler MessageHandler, opts *SubscribeOptions) (*Subscriber, error) {
	ctx, cancel := context.WithCancel(ctx)

	poolOpts := ants.Options{
		ExpiryDuration:   time.Hour,
		PreAlloc:         true,
		MaxBlockingTasks: opts.WorkerCount * 2,
		Nonblocking:      true,
		PanicHandler: func(i interface{}) {
			logger.Error.Printf("Worker panic: %v\n", i)
		},
	}

	pool, err := ants.NewPool(opts.WorkerCount, ants.WithOptions(poolOpts))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create worker pool: %w", err)
	}

	sub := &Subscriber{
		connManager:     connManager,
		handler:         handler,
		opts:            opts,
		ctx:             ctx,
		cancel:          cancel,
		channelManagers: make([]*ChannelManager, opts.WorkerCount),
		pool:            pool,
		msgChan:         make(chan *amqp.Delivery, opts.MessageBuffer),
	}

	for i := 0; i < opts.WorkerCount; i++ {
		sub.channelManagers[i] = NewChannelManager(ctx, connManager)
	}

	return sub, nil
}

func (s *Subscriber) declareQueue(name string, workerID int, isRPC bool, config *QueueConfig) (*amqp.Queue, error) {
	ch, err := s.channelManagers[workerID].GetChannel()
	queueName := name

	if err != nil || ch == nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	err = ch.Qos(
		s.opts.PrefetchCount,
		0,
		false,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	if config == nil {
		config = DefaultQueueConfig()
		if isRPC {
			config.AutoDelete = true
		}
	}

	reply, err := ch.QueueDeclare(
		queueName,
		config.Durable,
		config.AutoDelete,
		config.Exclusive,
		config.NoWait,
		config.Args,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	return &reply, nil
}

func (s *Subscriber) Start() error {
	if s.isRunning.Swap(true) {
		return fmt.Errorf("subscriber is already running")
	}
	for i := 0; i < s.opts.WorkerCount; i++ {
		s.wg.Add(1)
		workerID := i
		go func(workerID int) {
			err := s.pool.Submit(func() {
				s.runWorker(workerID)
			})
			if err != nil {
				return
			}
		}(workerID)
	}

	return nil
}

func (s *Subscriber) runWorker(workerID int) {
	defer s.wg.Done()

	backoff := &exponentialBackoff{
		min:    100 * time.Millisecond,
		max:    30 * time.Second,
		factor: 2,
	}

	for s.isRunning.Load() {
		logger.Debug.Println("Worker", workerID, "consuming...")
		if err := s.consume(workerID); err != nil {
			logger.Error.Printf("Worker %d consume error: %v\n", workerID, err)
			backoff.sleep()
			continue
		}
		backoff.reset()
	}
}

type exponentialBackoff struct {
	min    time.Duration
	max    time.Duration
	factor float64
	curr   time.Duration
}

func (b *exponentialBackoff) sleep() {
	if b.curr == 0 {
		b.curr = b.min
	} else {
		b.curr = time.Duration(float64(b.curr) * b.factor)
		if b.curr > b.max {
			b.curr = b.max
		}
	}
	time.Sleep(b.curr)
}

func (b *exponentialBackoff) reset() {
	b.curr = 0
}

func (s *Subscriber) consume(workerID int) error {
	ch, err := s.channelManagers[workerID].GetChannel()
	if err != nil || ch == nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}

	q, err := s.declareQueue(s.opts.QueueName, workerID, s.opts.IsRPC, s.opts.QueueOpts)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	consumerName := fmt.Sprintf("%s-%d-%d", s.opts.ConsumerName, workerID, time.Now().Unix())
	msgs, err := ch.Consume(
		q.Name,
		consumerName,
		s.opts.AutoAck,
		s.opts.Exclusive,
		s.opts.NoLocal,
		s.opts.NoWait,
		s.opts.Args,
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming %d: %w", workerID, err)
	}

	for msg := range msgs {
		select {
		case <-s.ctx.Done():
			logger.Error.Printf("Worker %d stopping\n", workerID)
			return nil
		default:
			if err := s.processMessage(workerID, &msg); err != nil {
				logger.Error.Printf("Worker %d failed to process message: %v\n", workerID, err)
			}
		}
	}

	return fmt.Errorf("consume channel closed")
}

func (s *Subscriber) processMessage(workerID int, msg *amqp.Delivery) error {
	response, err := s.handler(msg)

	if err != nil {
		if msg.CorrelationId != "" {
			payload, er := NewMessage(&_type.Response{
				Code:    http.StatusInternalServerError,
				Message: "Error when handler message",
				Error:   err,
			}, &msg.Headers)
			if er != nil {
				return fmt.Errorf("failed to create error payload: %w", err)
			}
			if sendErr := s.sendReply(workerID, msg, payload); sendErr != nil {
				return fmt.Errorf("failed to send error reply: %w", sendErr)
			}
		}
		if err = msg.Reject(true); err != nil {
			return fmt.Errorf("failed to reject message: %w", err)
		}
		return fmt.Errorf("handler error: %w", err)
	} else if msg.CorrelationId != "" {
		payload, err := NewMessage(response, &msg.Headers)
		if err != nil {
			return fmt.Errorf("failed to create response payload: %w", err)
		}
		if sendErr := s.sendReply(workerID, msg, payload); sendErr != nil {
			return fmt.Errorf("failed to send reply: %w", sendErr)
		}
	}

	if !s.opts.AutoAck {
		if err := msg.Ack(false); err != nil {
			return fmt.Errorf("failed to acknowledge message: %w", err)
		}
	}

	return nil
}

func (s *Subscriber) sendReply(workerID int, delivery *amqp.Delivery, msg *Message) error {
	s.mu.RLock()
	ch, err := s.channelManagers[workerID].GetChannel()
	s.mu.RUnlock()

	if err != nil || ch == nil {
		return fmt.Errorf("channel not available")
	}

	var msgHeaders amqp.Table

	err = helper.JSONToStruct[amqp.Table](delivery.Headers, &msgHeaders)
	if err != nil {
		return fmt.Errorf("failed to convert headers to map: %w", err)
	}

	msg.Headers = msgHeaders
	payload := msg.GenerateRPCReplyPayload(delivery.CorrelationId)

	if msgHeaders["reply_to"] == nil {
		return fmt.Errorf("reply_to header not found")
	}

	err = ch.Publish(
		"",
		msgHeaders["reply_to"].(string),
		false,
		false,
		*payload)
	if err != nil {
		return fmt.Errorf("failed to send reply: %w", err)
	}
	return nil
}

func (s *Subscriber) Stop() error {
	if !s.isRunning.Swap(false) {
		return nil
	}

	s.cancel()

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second * 60):
		return fmt.Errorf("timeout waiting for workers to stop")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, ch := range s.channelManagers {
		if ch != nil {
			if err := ch.Close(); err != nil {
				logger.Error.Printf("Error closing channel for worker %d: %v\n", i, err)
			}
			s.channelManagers[i] = nil
		}
	}

	close(s.msgChan)
	s.pool.Release()
	return nil
}

func (s *Subscriber) GetRunningWorkers() int {
	return s.pool.Running()
}

func (s *Subscriber) GetWorkerCapacity() int {
	return s.pool.Cap()
}

func (s *Subscriber) IsHealthy() bool {
	return s.isRunning.Load() && s.pool.Running() > 0
}
