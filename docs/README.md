![Rump](img/logo.svg?sanitize=true)

[![Go Report Card](https://goreportcard.com/badge/github.com/stickermule/rump)](https://goreportcard.com/report/github.com/stickermule/rump)
[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://godoc.org/github.com/stickermule/rump)
[![CI](https://img.shields.io/badge/master-pass-green.svg)](https://github.com/stickermule/rump/commits/master)

Hot sync two Redis databases using dumps.

## Why

There's no easy way to sync data from an [AWS ElastiCache](https://docs.aws.amazon.com/AmazonElastiCache/latest/red-ug/RestrictedCommands.html) or [GCP MemoryStore](https://cloud.google.com/memorystore/docs/reference/redis-configs#blocked) Redis cluster; the standard commands `BGSAVE` and `SLAVEOF` are blocked.

Rump is able to live sync Redis databases across cloud providers by only using `SCAN`, `DUMP` and `RESTORE`.

It's used at [Sticker Mule](https://www.stickermule.com) to keep staging and development environments in sync with the production AWS/GCP Redis clusters.

## Examples

```sh
# Sync local Redis DB 1 to DB 2.
$ rump -from redis://127.0.0.1:6379/1 -to redis://127.0.0.1:6379/2

# Sync ElastiCache cluster to local.
$ rump -from redis://production.cache.amazonaws.com:6379/1 -to redis://127.0.0.1:6379/1

# Sync protected ElastiCache via EC2 port forwarding.
$ ssh -L 6969:production.cache.amazonaws.com:6379 -N username@xxx.xxx.xxx.xxx &
$ rump -from redis://127.0.0.1:6969/1 -to redis://127.0.0.1:6379/1

# Dump GCP MemoryStore to file.
$ rump -from redis://10.0.20.2:6379/1 -to /backup/memorystore.rump

# Restore backup to ElastiCache.
$ rump -from /backup/memorystore.rump -to redis://production.cache.amazonaws.com:6379/1

# Sync with verbose mode disabled.
$ rump -from redis://127.0.0.1:6379/1 -to redis://127.0.0.1:6379/2 -silent

# Sync with TTLs.
$ rump -from redis://127.0.0.1:6379/1 -to redis://127.0.0.1:6379/2 -ttl
```

## Features

- Uses `SCAN` instead of `KEYS` to avoid DoS servers.
- Doesn't use any temp file.
- Can sync any key type.
- Can optionally sync TTLs.
- Uses buffered channels to optimize slow source servers.
- Uses implicit pipelining to minimize network roundtrips.
- Supports two-step sync: dump source to file, restore file to database.
- Supports Redis URIs with auth.
- Offers the same guarantees of the [SCAN](https://redis.io/commands/scan#scan-guarantees) command.

## Demo

[![asciicast](https://asciinema.org/a/255784.png)](https://asciinema.org/a/255784)

## Development

```sh
# requirements: docker, docker-compose (dc)
dc up # watch/run Rump tests and a Redis container
dc run --rm rump sh # get shell on Rump container
dc run --rm redis sh; redis-cli -h redis # get Redis console
```

## Install

Binaries can be found on the [releases](https://github.com/stickermule/rump/releases) page.

```
curl -SL https://github.com/stickermule/rump/releases/download/1.0.0/rump-1.0.0-linux-amd64 -o rump \
  && chmod +x rump;
./rump
```
You can run `rump` in a container following the [Dockerfile.example](/infra/Dockerfile.example).

## Mentions

- [Sticker Mule Blog](https://www.stickermule.com/blog/introducing-rump)
- [The Changelog](http://email.changelog.com/t/ViewEmail/t/13CBF627BB99BB74/)
- [Go Weekly](http://golangweekly.com/issues/138)
- [DB Weekly](http://dbweekly.com/issues/132)
- [Cron Weekly](https://www.cronweekly.com/issue-59/)
- [Hacker News](https://news.ycombinator.com/front?day=2016-12-05&p=2)
- Redis Weekly

## Maintainers

[nixtrace](https://github.com/nixtrace)

## Collaboration & License
- [Contributing](CONTRIBUTING.md)
- [Code of Conduct](CONTRIBUTING.md)
- [MIT License](https://opensource.org/licenses/MIT)
