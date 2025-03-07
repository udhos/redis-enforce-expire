[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/redis-enforce-expire/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/redis-enforce-expire)](https://goreportcard.com/report/github.com/udhos/redis-enforce-expire)
[![Go Reference](https://pkg.go.dev/badge/github.com/udhos/redis-enforce-expire.svg)](https://pkg.go.dev/github.com/udhos/redis-enforce-expire)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/redis-enforce-expire)](https://artifacthub.io/packages/search?repo=redis-enforce-expire)
[![Docker Pulls](https://img.shields.io/docker/pulls/udhos/redis-enforce-expire)](https://hub.docker.com/r/udhos/redis-enforce-expire)

# redis-enforce-expire

[redis-enforce-expire](https://github.com/udhos/redis-enforce-expire) ensures all redis keys have expiration defined.

# usage

See rules in [./rules.yaml](./rules.yaml).

Run like this:

```
$ redis-enforce-expire
```

Look at this log:

```
2025/03/07 01:28:37 INFO setExpire: rule-01[1/1]db:0 dry=false concurrent=true total_keys=0 ttl_errors=0 missing_ttl=0(1m0s) clamped_ttl=0(5m0s) expire_errors=0 elapsed=86.412796ms
```

Description:

```
rule-01[1/1]db:0 = <ruleName>[<ruleNumber>/<totalRules>]db:<dbNumber>
dry              = running in dry mode
concurrent       = running in concurrent mode (multiple goroutines)
total_keys       = total number of keys found
ttl_errors       = number of errors while getting TTL for keys
missing_ttl      = number of keys without TTL (these keys expirations were set to default_ttl)
clamped_ttl      = number of keys with TTL below max_ttl (these keys expirations were set to max_ttl)
elapsed          = time spent processing this DB
```
