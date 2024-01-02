package reids

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/ssh"
	"net"
	"os"
	"time"
)

const (
	SSHKeyTypeKey      SSHKeyType = "KEY"
	SSHKeyTypePassword SSHKeyType = "PASSWORD"
)

type SSHKeyType = string

type SSHConfig struct {
	Host     string
	User     string
	Port     string
	KeyType  SSHKeyType
	Password string
	KeyFile  string
	TimeOut  time.Duration
}

func (sc *SSHConfig) dialWithPassword() (*ssh.Client, error) {
	if sc.TimeOut == 0 {
		sc.TimeOut = time.Second * 15
	}
	config := &ssh.ClientConfig{
		User: sc.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(sc.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         sc.TimeOut,
	}
	return ssh.Dial("tcp", net.JoinHostPort(sc.Host, sc.Port), config)
}

func (sc *SSHConfig) dialWithKeyFile() (*ssh.Client, error) {
	if sc.TimeOut == 0 {
		sc.TimeOut = time.Second * 15
	}

	config := &ssh.ClientConfig{
		User:            sc.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         sc.TimeOut,
	}
	if k, err := os.ReadFile(sc.KeyFile); err != nil {
		return nil, err
	} else {
		signer, err := ssh.ParsePrivateKey(k)
		if err != nil {
			return nil, err
		}
		config.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
	}
	return ssh.Dial("tcp", net.JoinHostPort(sc.Host, sc.Port), config)
}

type RedisConfig struct {
	Host     string
	Port     string
	DB       int
	Password string
}

type RedisClient struct {
	client    *redis.Client
	sshClient *ssh.Client
}

func (m *RedisClient) Client() *redis.Client {
	return m.client
}

func (m *RedisClient) Close() {
	if m.client != nil {
		m.client.Close()
	}
	if m.sshClient != nil {
		m.sshClient.Close()
	}
}

func NewRedisClient(redisC *RedisConfig, sshC *SSHConfig) (*RedisClient, error) {
	var (
		options   *redis.Options
		sshClient *ssh.Client
	)
	if sshC != nil {
		var err error
		switch sshC.KeyType {
		case SSHKeyTypeKey:
			sshClient, err = sshC.dialWithKeyFile()
		case SSHKeyTypePassword:
			sshClient, err = sshC.dialWithPassword()
		default:
			return nil, fmt.Errorf("unknown ssh type")
		}
		if err != nil {
			return nil, fmt.Errorf("ssh connect error: %w", err)
		}
		options = &redis.Options{
			Addr:     net.JoinHostPort(redisC.Host, redisC.Port),
			Password: redisC.Password, // 没有密码，默认值
			DB:       redisC.DB,       // 默认DB 0
			Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return sshClient.Dial(network, addr)
			},
			// SSH不支持超时设置，在这里禁用
			ReadTimeout:  -2,
			WriteTimeout: -2,
		}

	} else {
		options = &redis.Options{
			Addr:     net.JoinHostPort(redisC.Host, redisC.Port),
			Password: redisC.Password, // 没有密码，默认值
			DB:       redisC.DB,       // 默认DB 0
		}
	}

	rdb := redis.NewClient(options)
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}
	return &RedisClient{
		client:    rdb,
		sshClient: sshClient,
	}, nil
}
