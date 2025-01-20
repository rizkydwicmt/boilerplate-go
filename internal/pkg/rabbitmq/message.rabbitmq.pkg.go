package rabbitmq

import (
	"encoding/json"
	"fmt"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Message struct {
	ID          string     `json:"id"`
	Body        []byte     `json:"content"`
	Headers     amqp.Table `json:"headers,omitempty"`
	Timestamp   time.Time  `json:"timestamp"`
	ContentType string     `json:"content_type"`
}

func NewMessage(payload interface{}, headers *amqp.Table) (*Message, error) {
	gid, err := gonanoid.New()
	if err != nil {
		return nil, err
	}
	id := fmt.Sprintf("msg_%s_%d", gid, time.Now().Unix())

	var body []byte
	var contentType string
	switch v := payload.(type) {
	case string:
		body = []byte(v)
		contentType = "text/plain"
	case []byte:
		body = v
		contentType = "application/octet-stream"
	default:
		body, err = json.Marshal(v)
		if err != nil {
			return nil, err
		}
		contentType = "application/json"
	}

	if headers == nil {
		headers = &amqp.Table{}
	}

	return &Message{
		ID:          id,
		Body:        body,
		Headers:     *headers,
		Timestamp:   time.Now(),
		ContentType: contentType,
	}, nil
}

func (m *Message) GeneratePayload() *amqp.Publishing {
	m.Headers["id"] = m.ID

	return &amqp.Publishing{
		ContentType:  m.ContentType,
		Body:         m.Body,
		MessageId:    m.ID,
		Timestamp:    m.Timestamp,
		DeliveryMode: amqp.Persistent,
		Headers:      m.Headers,
	}
}

func (m *Message) GenerateRPCPayload(queueName, reply string) *amqp.Publishing {
	m.Headers["id"] = m.ID
	m.Headers["reply_to"] = reply
	m.Headers["correlation_id"] = m.ID

	return &amqp.Publishing{
		ContentType:   m.ContentType,
		Body:          m.Body,
		MessageId:     m.ID,
		Timestamp:     m.Timestamp,
		DeliveryMode:  amqp.Persistent,
		ReplyTo:       queueName,
		CorrelationId: m.ID,
		Headers:       m.Headers,
	}
}

func (m *Message) GenerateRPCReplyPayload(correlationID string) *amqp.Publishing {
	return &amqp.Publishing{
		ContentType:   m.ContentType,
		Body:          m.Body,
		Timestamp:     m.Timestamp,
		DeliveryMode:  amqp.Persistent,
		CorrelationId: correlationID,
		Headers:       m.Headers,
	}
}
