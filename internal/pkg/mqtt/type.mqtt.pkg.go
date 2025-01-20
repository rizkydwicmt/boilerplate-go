package mqtt

import (
	"boilerplate-go/internal/pkg/redis"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"golang.org/x/crypto/bcrypt"
)

type Client struct {
	client mqtt.Client
	redis  redis.IRedis
}

type Config struct {
	URL      string
	ClientID string
	Username string
	Password string
}

type IMqtt interface {
	Subscribe(topic string, qos byte, callback mqtt.MessageHandler)
	Publish(topic string, qos byte, retained bool, payload interface{}) error
	Disconnect(timeout uint)
	AddClient(clientKey *ClientKey, clientBody *ClientBody, expr time.Duration) error
	RemoveClient(clientKey *ClientKey) error
	ExtendTTLClient(clientKey *ClientKey) error
	Close()
}

type ClientKey struct {
	MountPoint string `json:"mount_point"`
	ClientID   string `json:"client_id"`
	Username   string `json:"username"`
}

type ACL struct {
	Pattern string `json:"pattern"`
}

type ClientBody struct {
	User         interface{} `json:"user"`
	Password     string      `json:"password"`
	SubscribeACL *[]ACL      `json:"subscribe_acl,omitempty"`
	PublishACL   *[]ACL      `json:"publish_acl,omitempty"`
}

type ClientBodyEncrypt struct {
	User         interface{} `json:"user"`
	Password     string      `json:"passhash"`
	SubscribeACL *[]ACL      `json:"subscribe_acl,omitempty"`
	PublishACL   *[]ACL      `json:"publish_acl,omitempty"`
}

func (cb *ClientBody) Encrypt() (*ClientBodyEncrypt, error) {
	pass, err := bcrypt.GenerateFromPassword([]byte(cb.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &ClientBodyEncrypt{
		User:         cb.User,
		Password:     string(pass),
		SubscribeACL: cb.SubscribeACL,
		PublishACL:   cb.PublishACL,
	}, nil
}
