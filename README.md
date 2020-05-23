# fast path of exile stash crawler

## Build

0. get golang version 1.14
1. git clone this repo
2. `go build .`

## Usage

Crawl garena PoE stashes:

```
./fastash web.poe.garena.tw
```

OR Crawl garena PoE stashes with proxies to speed up (example):

```
ssh -D 6666 user1@server1  # run in another terminal
ssh -D 6667 user2@server2  # run in another terminal
./fastash web.poe.garena.tw socks5://localhost:6666 socks5://localhost:6667
```

## Storage structure

Each `change_id` is stored in `stashes/${group}/${change_id}.json.gz` as gzipped-JSON. `${group}` equals sum of `${change_id}` divided by 10^8.
