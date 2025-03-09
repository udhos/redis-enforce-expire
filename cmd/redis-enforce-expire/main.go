// Package main implements the tool.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	_ "github.com/KimMachineGun/automemlimit"
	"github.com/redis/go-redis/v9"
	"github.com/udhos/redis-enforce-expire/internal/redisclient"
	_ "go.uber.org/automaxprocs"
)

func getVersion(me string) string {
	return fmt.Sprintf("%s version=%s runtime=%s GOOS=%s GOARCH=%s GOMAXPROCS=%d",
		me, version, runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.GOMAXPROCS(0))
}

type application struct {
	rules []rule
}

func main() {

	var showVersion bool
	flag.BoolVar(&showVersion, "version", showVersion, "show version")
	flag.Parse()

	me := filepath.Base(os.Args[0])

	{
		v := getVersion(me)
		if showVersion {
			fmt.Println(v)
			return
		}
		slog.Info(v)
	}

	rulesFile := envString("RULES", "rules.yaml")

	rules, errRules := loadRules(rulesFile)
	if errRules != nil {
		fatalf("error reading rules file=%s: %v", rulesFile, errRules)
	}

	infof("loaded %d rules from %s", len(rules), rulesFile)

	app := &application{
		rules: rules,
	}

	run(app)
}

func run(app *application) {
	const me = "run"

	var wg sync.WaitGroup

	begin := time.Now()

	//
	// scan rules
	//
	for i, r := range app.rules {
		ruleName := r.label(i, len(app.rules))
		first, last, errDB := r.parseRedisDB()
		if errDB != nil {
			errorf("%s: %s: redis db parse error: %v", me, ruleName, errDB)
			continue
		}
		infof("%s: %s db: %d-%d (%d databases)",
			me, ruleName, first, last, last-first+1)

		//
		// scan rule databases
		//
		for db := first; db <= last; db++ {
			dbName := fmt.Sprintf("%sdb:%d", ruleName, db)
			if r.MultipleGoroutines {
				wg.Add(1)
				go func() {
					setExpire(r, dbName, db, true)
					wg.Done()
				}()
			} else {
				setExpire(r, dbName, db, false)
			}
		}

	}

	wg.Wait()

	elap := time.Since(begin)

	infof("%s: elap=%v", me, elap)
}

func setExpire(r rule, dbName string, db int, concurrent bool) {
	const me = "setExpire"

	begin := time.Now()

	redisClient := redisclient.New(r.RedisAddr,
		r.RedisPassword, r.ClientName, db, r.TLS, r.TLSInsecureSkipVerify)
	ctx := context.TODO()

	var cursor uint64
	var n int
	var getTTLErrors int
	var missingTTL int
	var expireErrors int
	var clampedTTL int
	var scans int
	for {
		scans++
		var keys []string
		var err error
		if r.ScanType == "" {
			keys, cursor, err = redisClient.Scan(ctx, cursor, r.ScanMatch, r.ScanCount).Result()
		} else {
			keys, cursor, err = redisClient.ScanType(ctx, cursor, r.ScanMatch, r.ScanCount, r.ScanType).Result()
		}
		if err != nil {
			if r.CrashOnScanError {
				fatalf("%s: %s: scan error crash_on_scan_error=%t: %v",
					me, dbName, r.CrashOnScanError, err)
			}
			errorf("%s: %s: scan error crash_on_scan_error=%t: %v",
				me, dbName, r.CrashOnScanError, err)
			break
		}

		for _, k := range keys {
			dur, errDur := redisClient.TTL(ctx, k).Result()
			if errDur != nil {
				getTTLErrors++
				continue
			}
			switch {
			case dur == -1:
				missingTTL++
				if ok := expire(ctx, redisClient, k, r.DefaultTTL, r.AddRandomTTL, r.DryRun); !ok {
					expireErrors++
				}
			case dur > r.MaxTTL:
				clampedTTL++
				if ok := expire(ctx, redisClient, k, r.MaxTTL, r.AddRandomTTL, r.DryRun); !ok {
					expireErrors++
				}
			}
		}

		n += len(keys)
		if cursor == 0 {
			break
		}
	}

	elap := time.Since(begin)
	infof("%s: %s dry=%t concurrent=%t scan_match=%s scan_count=%d scans=%d total_keys=%d ttl_errors=%d missing_ttl=%d(%v) clamped_ttl=%d(%v) expire_errors=%d elapsed=%v",
		me, dbName, r.DryRun, concurrent, r.ScanMatch, r.ScanCount, scans, n, getTTLErrors, missingTTL, r.DefaultTTL, clampedTTL, r.MaxTTL, expireErrors, elap)
}

func expire(ctx context.Context, redisClient *redis.Client, key string, dur, addRandomTTL time.Duration, dry bool) bool {
	if dry {
		return true
	}
	add := time.Duration(rand.Int64N(addRandomTTL.Nanoseconds() + 1))
	ok, errExpire := redisClient.Expire(ctx, key, dur+add).Result()
	return ok && errExpire == nil
}
