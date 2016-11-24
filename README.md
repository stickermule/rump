# Rump.

Sync two Redis servers using dumps.

`rump -from redis://1234.cache.amazonaws.com:6379/1 -to redis://127.0.0.1:6379/1`

## Why.

[ElastiCache]( http://docs.aws.amazon.com/AmazonElastiCache/latest/UserGuide/ClientConfig.RestrictedCommands.html ) doesn't support any of the standard Redis backup commands.

Rump is able to transfer keys from an ElastiCache cluster or any Redis server to another Redis server, by only using `SCAN`, `DUMP` and `RESTORE`.

## Features.

- Uses `SCAN` instead of `KEYS` to avoid DoS your own server.
- Can sync any key type.
- Drops the TTL on purpose, since it wouldn't be in sync.
- Doesn't use any temp file.
- Uses buffered channels to optimize slow source servers.
- Uses pipelines to minimize network roundtrips.

## Examples.

- Sync local Redis DB 1 to DB 2.
`rump -from redis://127.0.0.1:6379/1 -to redis://127.0.0.1:6379/2`

- Sync ElastiCache cluster to local.
`rump -from redis://production.cache.amazonaws.com:6379/1 -to redis://127.0.0.1:6379/1`

- Sync protected ElastiCache via EC2 port forwarding.
`$ ssh -L 6969:production.cache.amazonaws.com:6379 -N ubuntu@xxx.xxx.xxx.xxx &
rump -from redis://127.0.0.1:6969/1 -to redis://127.0.0.1:6379/1
`
