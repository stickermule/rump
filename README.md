<h1 align="center">
<img src="https://rawgit.com/jdorfman/rump/master/assets/images/rump_logo.svg">
</h1>
[![GoDoc](https://godoc.org/github.com/stickermule/rump?status.svg)](https://godoc.org/github.com/stickermule/rump)

Hot sync two Redis databases using dumps.

## Why

There's no easy way to get/sync data from an [AWS ElastiCache]( http://docs.aws.amazon.com/AmazonElastiCache/latest/UserGuide/ClientConfig.RestrictedCommands.html ) Redis cluster.

> **@bdq**: Hey, let's keep our staging Redis containers in sync with our AWS ElastiCache. `BGSAVE` and copy the .rdb?

>**@badshark**: Yeah, awesome, let me try... [Nope, not supported](http://docs.aws.amazon.com/AmazonElastiCache/latest/UserGuide/ClientConfig.RestrictedCommands.html).

>**@bdq**: Ah, that's bad. We'll have to set the containers as `SLAVEOF`?

>**@badshark**: That makes sense, doing it... [Nope, not supported](http://docs.aws.amazon.com/AmazonElastiCache/latest/UserGuide/ClientConfig.RestrictedCommands.html).

>**@bdq**: WAT. Let's use an open source tool to do the sync?

>**@badshark**: Most of them use `KEYS` to get the keys, we'd DoS our own server.

>**@bdq**: Let's write a script?

>**@badshark**: Tried. Bash doesn't like key dumps, Ruby/Python + deps take more space than Redis inside the container.

>**[@bdq](https://github.com/BDQ)** and **[@badshark](https://github.com/badshark)**: Let's write it in Go?


Rump is able to transfer keys from an ElastiCache cluster or any Redis server to another Redis server, by only using `SCAN`, `DUMP` and `RESTORE`.

## Features

- Uses `SCAN` instead of `KEYS` to avoid DoS your own server.
- Can sync any key type.
- Drops the TTL on purpose, since it wouldn't be in sync.
- Doesn't use any temp file.
- Uses buffered channels to optimize slow source servers.
- Uses pipelines to minimize network roundtrips.

## Examples

```sh
# Sync local Redis DB 1 to DB 2.
$ rump -from redis://127.0.0.1:6379/1 -to redis://127.0.0.1:6379/2

# Sync ElastiCache cluster to local.
$ rump -from redis://production.cache.amazonaws.com:6379/1 -to redis://127.0.0.1:6379/1

# Sync protected ElastiCache via EC2 port forwarding.
$ ssh -L 6969:production.cache.amazonaws.com:6379 -N ubuntu@xxx.xxx.xxx.xxx &
$ rump -from redis://127.0.0.1:6969/1 -to redis://127.0.0.1:6379/1
```

## Demo

[![asciicast](https://asciinema.org/a/94355.png)](https://asciinema.org/a/94355)

## Contribute

- We use GitHub issues to discuss everything: features, bugs, docs.
- Before sending a pull request always open an issue.

## Install

You can find pre-compiled binaries on the [releases](https://github.com/stickermule/rump/releases) page. If you don't see your OS/Arch there, just ask :)

## Mentions

- [Sticker Mule Blog](https://www.stickermule.com/blog/introducing-rump)
- [DB Weekly](http://dbweekly.com/issues/132)
- [Go Newsletter](http://golangweekly.com/issues/138)

## Maintainers

[badshark](https://github.com/badshark)

## License

Rump is licensed under the [MIT License](https://opensource.org/licenses/MIT)
