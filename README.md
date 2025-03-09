[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/redis-enforce-expire/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/redis-enforce-expire)](https://goreportcard.com/report/github.com/udhos/redis-enforce-expire)
[![Go Reference](https://pkg.go.dev/badge/github.com/udhos/redis-enforce-expire.svg)](https://pkg.go.dev/github.com/udhos/redis-enforce-expire)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/redis-enforce-expire)](https://artifacthub.io/packages/search?repo=redis-enforce-expire)
[![Docker Pulls](https://img.shields.io/docker/pulls/udhos/redis-enforce-expire)](https://hub.docker.com/r/udhos/redis-enforce-expire)

# redis-enforce-expire

[redis-enforce-expire](https://github.com/udhos/redis-enforce-expire) ensures all redis keys have expiration defined.

# rules

See rules in [./rules.yaml](./rules.yaml).

Example rule:

```yaml
- rule_name: rule1
  client_name: redis-enforce-expire
  dry_run: false
  redis_addr: "localhost:6379"
  redis_password: "123"
  redis_db: "0-1" # 0 to 1
  tls: false
  tls_insecure_skip_verify: false
  scan_match: "*"
  scan_count: 1000
  scan_type: "" # emtpy means all
  multiple_goroutines: true
  default_ttl: 1m
  max_ttl: 5m
  crash_on_scan_error: false
  add_random_ttl: 10s # randomize expiration to avoid expiring many keys at the same time
  pipeline_batch_size: 1000 # pipeline is disabled if batch is less than 1
```

Rule fields descriptions:

```bash
rule_name:                give a name for the rule
client_name:              set our redis client name for connections
dry_run:                  enable dry mode (if enabled, the redis server is not modified)
redis_addr:               redis server hostname:port
redis_password:           redis password. leave it empty if your redist server does not have a password
redis_db:                 range of DBs to scan over
tls:                      enable TLS
tls_insecure_skip_verify: disable TLS certificate verification on our side
scan_match:               key match pattern for SCAN
scan_count:               how many keys the SCAN command should retrieve for every request
scan_type:                retrict scan type or leave as empty for all scan types
multiple_goroutines:      enable multiple concurrent goroutines (one for each db)
default_ttl:              any key without TTL will get its expire set to default_ttl
max_ttl:                  any TTL found above max_ttl will be clamped down to max_ttl
crash_on_scan_error:      enable this to make the program crash on first scan error
add_random_ttl:           randomize expiration to avoid expiring many keys at the same time
pipeline_batch_size:      pipeline batch size (how many commands to send before receiving responses)
```

# running

Run like this:

```bash
$ redis-enforce-expire
```

Look at this example log line:

```bash
2025/03/09 02:16:50 INFO setExpire: rule-01[1/1]db:0 dry=false concurrent=true pipeline_batch=1000 scan_match=* scan_count=1000 scans=1000 total_keys=1000000 ttl_errors=0 missing_ttl=1000000(1m0s) clamped_ttl=0(5m0s) expire_errors=0 exec_errors=0 elapsed=1.995328952s
```

Log fields description:

```bash
rule-01[1/1]db:0 = <ruleName>[<ruleNumber>/<totalRules>]db:<dbNumber>
dry              = running in dry mode
concurrent       = running in concurrent mode (multiple goroutines)
pipeline_batch   = pipeline batch size (if less than 1, pipeline is disabled)
scan_match       = scan match pattern for keys
scan_count       = scan count (scan batch size)
scans            = number of scans executed
total_keys       = total number of keys found
ttl_errors       = number of errors while getting TTL for keys
missing_ttl      = number of keys without TTL (these keys expirations were set to default_ttl)
clamped_ttl      = number of keys with TTL below max_ttl (these keys expirations were set to max_ttl)
expire_errors    = errors setting expire/TTL on keys
exec_errors      = pipeline execution errors
elapsed          = time spent processing this DB
```
