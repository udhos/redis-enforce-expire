package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type rule struct {
	RuleName              string        `yaml:"rule_name"`
	ClientName            string        `yaml:"client_name"`
	DryRun                bool          `yaml:"dry_run"`
	RedisAddr             string        `yaml:"redis_addr"`
	RedisPassword         string        `yaml:"redis_password"`
	RedisDB               string        `yaml:"redis_db"`
	TLS                   bool          `yaml:"tls"`
	TLSInsecureSkipVerify bool          `yaml:"tls_insecure_skip_verify"`
	ScanMatch             string        `yaml:"scan_match"`
	ScanCount             int64         `yaml:"scan_count"`
	ScanType              string        `yaml:"scan_type"`
	MultipleGoroutines    bool          `yaml:"multiple_goroutines"`
	DefaultTTL            time.Duration `yaml:"default_ttl"`
	MaxTTL                time.Duration `yaml:"max_ttl"`
	CrashOnScanError      bool          `yaml:"crash_on_scan_error"`
	AddRandomTTL          time.Duration `yaml:"add_random_ttl"`
}

func (r *rule) label(index, total int) string {
	return fmt.Sprintf("%s[%d/%d]", r.RuleName, index+1, total)
}

func (r *rule) parseRedisDB() (int, int, error) {
	const me = "rule.parseRedisDB"
	db := r.RedisDB
	before, after, found := strings.Cut(db, "-")
	first, errFirst := strconv.Atoi(before)
	if errFirst != nil {
		return 0, 0, fmt.Errorf("%s: error parsing first db from '%s': %v", me, db, errFirst)
	}
	last := first
	if found {
		j, errLast := strconv.Atoi(after)
		if errLast != nil {
			return 0, 0, fmt.Errorf("%s: error parsing last db from '%s': %v", me, db, errLast)
		}
		last = j
	}
	if first > last {
		return 0, 0, fmt.Errorf("%s: first=%d > last=%d", me, first, last)
	}
	return first, last, nil
}

func loadRules(path string) ([]rule, error) {
	data, errRead := os.ReadFile(path)
	if errRead != nil {
		return nil, errRead
	}
	return newRules(data)
}

const (
	defaultScanCount = 10
	defaultRedisDB   = "0-15"
)

func newRules(data []byte) ([]rule, error) {
	const me = "newRules"
	var rules []rule

	if errYaml := yaml.Unmarshal(data, &rules); errYaml != nil {
		return nil, errYaml
	}

	if len(rules) < 1 {
		return nil, errors.New("empty rules list")
	}

	for i, r := range rules {
		if r.RuleName == "" {
			r.RuleName = fmt.Sprintf("rule-%02d", i+1)
		}
		if r.RedisDB == "" {
			infof("%s: %s: redis_db='%s', forcing to default=%s",
				me, r.label(i, len(rules)), r.RedisDB, defaultRedisDB)
			r.RedisDB = defaultRedisDB
		}
		if r.ScanCount < 1 {
			infof("%s: %s: scan_count=%d, forcing to default=%d",
				me, r.label(i, len(rules)), r.ScanCount, defaultScanCount)
			r.ScanCount = defaultScanCount
		}
		rules[i] = r
	}

	return rules, nil
}
