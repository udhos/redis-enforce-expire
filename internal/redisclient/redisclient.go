// Package redisclient creates a redis client.
package redisclient

import (
	"crypto/tls"

	"github.com/redis/go-redis/v9"
)

// New creates a redis client.
func New(addr, password, clientName string, db int, useTLS,
	tlsInsecureSkipVerify bool) *redis.Client {
	redisOptions := &redis.Options{
		Addr:       addr,
		Password:   password,
		DB:         db,
		ClientName: clientName,
	}

	if useTLS || tlsInsecureSkipVerify {
		redisOptions.TLSConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: tlsInsecureSkipVerify,
		}
	}

	return redis.NewClient(redisOptions)
}
