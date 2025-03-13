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
	"sync"
	"time"

	_ "github.com/KimMachineGun/automemlimit"
	"github.com/redis/go-redis/v9"
	"github.com/udhos/boilerplate/awsconfig"
	"github.com/udhos/boilerplate/boilerplate"
	"github.com/udhos/boilerplate/secret"
	"github.com/udhos/redis-enforce-expire/internal/redisclient"
	_ "go.uber.org/automaxprocs"
)

type application struct {
	rules []rule
}

func main() {

	var showVersion bool
	flag.BoolVar(&showVersion, "version", showVersion, "show version")
	flag.Parse()

	me := filepath.Base(os.Args[0])

	{
		v := boilerplate.LongVersion(me + " version=" + version)
		if showVersion {
			fmt.Println(v)
			return
		}
		slog.Info(v)
	}

	secretDebug := envBool("SECRET_DEBUG", false)

	sec := initSecret(me, secretDebug)

	rulesFile := envString("RULES", "rules.yaml")
	logRules := envBool("LOG_RULES", true)

	rules, errRules := loadRules(rulesFile, sec, logRules, secretDebug)
	if errRules != nil {
		fatalf("error reading rules file=%s: %v", rulesFile, errRules)
	}

	infof("loaded %d rules from %s", len(rules), rulesFile)

	app := &application{
		rules: rules,
	}

	run(app)
}

func initSecret(me string, secretDebug bool) *secret.Secret {
	roleArn := envString("ROLE_ARN", "")

	awsConfOptions := awsconfig.Options{
		RoleArn:         roleArn,
		RoleSessionName: me,
	}

	secretOptions := secret.Options{
		AwsConfigSource: &secret.AwsConfigSource{AwsConfigOptions: awsConfOptions},
		Debug:           secretDebug,
	}
	secret := secret.New(secretOptions)
	return secret
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

	pipeline := r.PipelineBatchSize > 0
	var p pipe
	var execErrors int
	if pipeline {
		p = pipe{
			redisClient:  redisClient,
			batchMax:     r.PipelineBatchSize,
			maxTTL:       r.MaxTTL,
			defaultTTL:   r.DefaultTTL,
			addRandomTTL: r.AddRandomTTL,
			dry:          r.DryRun,
		}
	}

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

		if pipeline {
			//
			// pipelined
			//
			for _, k := range keys {
				ttl, exec, missing, clamped, expire := p.add(ctx, k)
				getTTLErrors += ttl
				execErrors += exec
				missingTTL += missing
				clampedTTL += clamped
				expireErrors += expire
			}
		} else {
			//
			// no pipeline
			//
			for _, k := range keys {
				dur, errDur := redisClient.TTL(ctx, k).Result()
				if errDur != nil {
					getTTLErrors++
					continue
				}
				switch evalDur(dur, r.MaxTTL+r.AddRandomTTL) {
				case shouldExpireDefault:
					missingTTL++
					if ok := expire(ctx, redisClient, k, r.DefaultTTL, r.AddRandomTTL, r.DryRun); !ok {
						expireErrors++
					}
				case shouldClamp:
					clampedTTL++
					if ok := expire(ctx, redisClient, k, r.MaxTTL, r.AddRandomTTL, r.DryRun); !ok {
						expireErrors++
					}
				}
			}
		}

		n += len(keys)
		if cursor == 0 {
			break
		}
	}

	if pipeline {
		ttl, exec, missing, clamped, expire := p.closeBatch(ctx)
		getTTLErrors += ttl
		execErrors += exec
		missingTTL += missing
		clampedTTL += clamped
		expireErrors += expire
	}

	elap := time.Since(begin)
	infof("%s: %s dry=%t concurrent=%t pipeline_batch=%d scan_match=%s scan_count=%d scans=%d total_keys=%d ttl_errors=%d missing_ttl=%d(%v) clamped_ttl=%d(%v) expire_errors=%d exec_errors=%d elapsed=%v",
		me, dbName, r.DryRun, concurrent, r.PipelineBatchSize, r.ScanMatch, r.ScanCount, scans, n, getTTLErrors, missingTTL, r.DefaultTTL, clampedTTL, r.MaxTTL, expireErrors, execErrors, elap)
}

const (
	shouldIgnore        = 0
	shouldExpireDefault = 1
	shouldClamp         = 2
)

func evalDur(dur, maxTTL time.Duration) int {
	switch {
	case dur == -1:
		return shouldExpireDefault
	case dur > maxTTL:
		return shouldClamp
	}
	return shouldIgnore
}

func expire(ctx context.Context, redisClient *redis.Client, key string, dur, addRandomTTL time.Duration, dry bool) bool {
	if dry {
		return true
	}
	add := randomDur(addRandomTTL)
	ok, errExpire := redisClient.Expire(ctx, key, dur+add).Result()
	return ok && errExpire == nil
}

func randomDur(delta time.Duration) time.Duration {
	return time.Duration(rand.Int64N(delta.Nanoseconds() + 1))
}

type pipe struct {
	pipe         redis.Pipeliner
	redisClient  *redis.Client
	batchMax     int
	batchCurrent int
	pipeExpire   redis.Pipeliner
	maxTTL       time.Duration
	defaultTTL   time.Duration
	addRandomTTL time.Duration
	dry          bool
}

func (p *pipe) add(ctx context.Context, key string) (getTTLErrors, execErrors,
	missingTTL, clampedTTL, expireErrors int) {
	if p.pipe == nil {
		p.pipe = p.redisClient.Pipeline()
	}
	p.pipe.TTL(ctx, key)
	p.batchCurrent++
	if p.batchCurrent >= p.batchMax {
		ttl, exec, missing, clamped, expire := p.closeBatch(ctx)
		getTTLErrors += ttl
		execErrors += exec
		missingTTL += missing
		clampedTTL += clamped
		expireErrors += expire
	}
	return
}

func (p *pipe) closeBatch(ctx context.Context) (getTTLErrors, execErrors,
	missingTTL, clampedTTL, expireErrors int) {

	defer func() {
		p.pipe = nil
		p.batchCurrent = 0
	}()

	if p.batchCurrent < 1 {
		return
	}
	cmds, errExec := p.pipe.Exec(ctx)
	if errExec != nil {
		execErrors++
		return
	}

	for _, cmd := range cmds {
		c := cmd.(*redis.DurationCmd)
		dur, errDur := c.Result()
		if errDur != nil {
			getTTLErrors++
			continue
		}

		key, errKey := getCmdKey(c)
		if errKey != nil {
			errorf("closeBatch: getCmdKey: %v", errKey)
			getTTLErrors++
			return
		}

		switch evalDur(dur, p.maxTTL+p.addRandomTTL) {
		case shouldExpireDefault:
			missingTTL++
			p.expire(ctx, key, p.defaultTTL, p.addRandomTTL, p.dry)
		case shouldClamp:
			clampedTTL++
			p.expire(ctx, key, p.maxTTL, p.addRandomTTL, p.dry)
		}
	}

	exec, expire := p.execExpire(ctx)
	execErrors += exec
	expireErrors += expire

	return
}

func getCmdKey(cmd redis.Cmder) (string, error) {
	args := cmd.Args()
	if len(args) < 2 {
		return "", fmt.Errorf("short command: len=%d: %v", len(args), args)
	}
	k := args[1]
	key, isStr := k.(string)
	if !isStr {
		return "", fmt.Errorf("command key not a string: type=%T value=%v args=%v",
			k, k, args)
	}
	return key, nil
}

// expire batches one pipelined expire command.
func (p *pipe) expire(ctx context.Context, key string, ttl, random time.Duration, dry bool) {
	if p.pipeExpire == nil {
		p.pipeExpire = p.redisClient.Pipeline()
	}
	dur := ttl + randomDur(random)
	if dry {
		return
	}
	p.pipeExpire.Expire(ctx, key, dur)
}

// execExpire executes batched expire commands.
func (p *pipe) execExpire(ctx context.Context) (execErrors, expireErrors int) {

	if p.pipeExpire == nil {
		return
	}

	defer func() {
		p.pipeExpire = nil
	}()

	cmds, errExec := p.pipeExpire.Exec(ctx)
	if errExec != nil {
		execErrors++
		return
	}
	for _, cmd := range cmds {
		c := cmd.(*redis.BoolCmd)
		ok, errExpire := c.Result()
		if !ok || errExpire != nil {
			expireErrors++
		}
	}
	return
}
