// Package main implements the tool.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/udhos/redis-enforce-expire/internal/redisclient"
)

func main() {
	var addr string
	var password string
	var keyPrefix string
	var db int
	var useTLS bool
	var tlsInsecureSkipVerify bool
	var pipeline bool
	var exitOnError bool
	var logErrors bool
	var pipelineBatchSize int

	var count int
	flag.IntVar(&count, "count", 1_000_000, "how many keys to create")
	flag.IntVar(&db, "db", 0, "select database")
	flag.IntVar(&pipelineBatchSize, "pipelineBatchSize", 10_000, "pipeline batch size")
	flag.StringVar(&addr, "addr", "localhost:6379", "redis host:port")
	flag.StringVar(&password, "password", "", "redis password")
	flag.StringVar(&keyPrefix, "keyPrefix", "test_", "key prefix")
	flag.BoolVar(&useTLS, "tls", false, "enable TLS")
	flag.BoolVar(&tlsInsecureSkipVerify, "tlsInsecureSkipVerify", false, "skip tls secure verify")
	flag.BoolVar(&pipeline, "pipeline", true, "redis pipelining")
	flag.BoolVar(&exitOnError, "exitOnError", true, "exit on error")
	flag.BoolVar(&logErrors, "logErrors", false, "log errors")
	flag.Parse()

	begin := time.Now()

	redisClient := redisclient.New(addr,
		password, "redis-create-keys", db, useTLS, tlsInsecureSkipVerify)
	ctx := context.TODO()

	var setCount int
	var setErrors int
	var pipelineBatches int

	var pipelineLabel string
	if pipeline {
		pipelineLabel = "pipelined"
	} else {
		pipelineLabel = "nopipe"
	}

	set := func(f func() error) {
		errSet := f()
		if errSet != nil {
			setErrors++
			if exitOnError {
				fatalf("%s (exitOnError) set %d error: %v",
					pipelineLabel, setCount, errSet)
			}
			if logErrors {
				errorf("%s (logErrors) set %d error: %v",
					pipelineLabel, setCount, errSet)
			}
		}
	}

	if pipeline {
		var i int
		for i < count {
			pipelineBatches++
			pipe := redisClient.Pipeline()
			for range pipelineBatchSize {
				if i >= count {
					break
				}
				set(func() error {
					setCount++
					str := genKey(keyPrefix, setCount)
					_, err := pipe.Set(ctx, str, setCount, 0).Result()
					return err
				})
				i++
			}
			_, errExec := pipe.Exec(ctx)
			if errExec != nil {
				if exitOnError {
					fatalf("%s (exitOnError) exec error: %v",
						pipelineLabel, errExec)
				}
				if logErrors {
					errorf("%s (logErrors) exec error: %v",
						pipelineLabel, errExec)
				}
			}
		}
	} else {
		for range count {
			set(func() error {
				setCount++
				str := genKey(keyPrefix, setCount)
				_, err := redisClient.Set(ctx, str, setCount, 0).Result()
				return err
			})
		}
	}

	elapsed := time.Since(begin)

	slog.Info(fmt.Sprintf("%s batches=%d keys=%d errors=%d elapsed=%v",
		pipelineLabel, pipelineBatches, setCount, setErrors, elapsed))
}

func genKey(prefix string, i int) string {
	return fmt.Sprintf("%s%d", prefix, i)
}

func fatalf(format string, a ...any) {
	slog.Error("FATAL: " + fmt.Sprintf(format, a...))
	os.Exit(1)
}

func errorf(format string, a ...any) {
	slog.Error(fmt.Sprintf(format, a...))
}
