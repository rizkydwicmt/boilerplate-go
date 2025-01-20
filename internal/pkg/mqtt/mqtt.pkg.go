package mqtt

import (
	"boilerplate-go/internal/pkg/logger"
	"boilerplate-go/internal/pkg/redis"
	"encoding/json"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func Setup(config *Config, rds redis.IRedis) (IMqtt, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.URL)
	opts.SetClientID(config.ClientID)
	opts.SetUsername(config.Username)
	opts.SetPassword(config.Password)
	opts.OnConnect = func(client mqtt.Client) {
		fmt.Println("Connected to IMqtt broker")
	}
	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		logger.Error.Printf("Connection lost: %v\n", err)
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		logger.Error.Fatalf("Failed to connect to broker: %v\n", token.Error())
		return nil, token.Error()
	}

	return &Client{
		client,
		rds,
	}, nil
}

func (m *Client) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) {
	if token := m.client.Subscribe(topic, qos, callback); token.Wait() && token.Error() != nil {
		logger.Error.Printf("Failed to subscribe to topic: %v\n", token.Error())
	}
}

func (m *Client) Publish(topic string, qos byte, retained bool, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	token := m.client.Publish(topic, qos, retained, data)
	token.Wait()
	return nil
}

func (m *Client) Disconnect(timeout uint) {
	m.client.Disconnect(timeout)
}

func (m *Client) AddClient(clientKey *ClientKey, clientBody *ClientBody, expr time.Duration) error {
	encBody, err := clientBody.Encrypt()
	if err != nil {
		return err
	}
	err = m.redis.Set(fmt.Sprintf(`[%q,%q,%q]`, clientKey.MountPoint, clientKey.ClientID, clientKey.Username), encBody, expr)
	if err != nil {
		return err
	}
	return nil
}

func (m *Client) ExtendTTLClient(clientKey *ClientKey) error {
	err := m.redis.Expire(fmt.Sprintf(`[%q,%q,%q]`, clientKey.MountPoint, clientKey.ClientID, clientKey.Username), 24*time.Hour)
	if err != nil {
		return err
	}
	return nil
}

func (m *Client) RemoveClient(clientKey *ClientKey) error {
	err := m.redis.Del(fmt.Sprintf(`[%q,%q,%q]`, clientKey.MountPoint, clientKey.ClientID, clientKey.Username))
	if err != nil {
		return err
	}
	return nil
}

func (m *Client) Close() {
	m.client.Disconnect(250)
}
