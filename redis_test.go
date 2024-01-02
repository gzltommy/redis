package reids

import (
	"context"
	"fmt"
	"testing"
)

func TestRedis(t *testing.T) {
	redisClient, err := NewRedisClient(
		&RedisConfig{
			Host:     "127.0.0.1",
			Port:     "6379",
			DB:       0,
			Password: "123",
		},
		&SSHConfig{
			Host:    "127.0.0.1",
			User:    "ubuntu",
			Port:    "22",
			KeyFile: `C:\Users\ww\.ssh\id_rsa`,
			KeyType: SSHKeyTypeKey,
		},
	)
	if err != nil {
		fmt.Printf("redis connect error: %s", err.Error())
		return
	}

	val, err := redisClient.Client().Do(context.Background(), "keys", "db_service_cache_user_1029007").Result()
	t.Log(err)
	t.Log(val)
}
