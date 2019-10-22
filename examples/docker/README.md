# Embed http-broacast in a Varnish image

This example shows how to embed a `http-broadcast` inside a varnish image to
automatically broadcast the PURGE requests to all the varnish containers.

## What's inside

- a varnish image containing a varnishd server and an `http-broadcast` instance
- a dummy `nginx` instance
- a mercure hub
- a `bench` container that send a `GET` request every seconds, and send a `PURGE` request every 10 seconds

## How to demo

Run the containers and watch the logs

```bash
$ docker-compose up --scale varnish=3
```

sample output

```
nginx_1    | 172.26.0.5 - - [15/Sep/2019:16:26:55 +0000] "GET / HTTP/1.1" 200 612 "-" "curl/7.65.1" "172.26.0.3"        <-- nginx is called because of cache missed of varnish
bench_1    | Age: 0                                                                                                     <-- bench receives a 0 sec old response
bench_1    | Age: 1
nginx_1    | 172.26.0.7 - - [15/Sep/2019:16:26:57 +0000] "GET / HTTP/1.1" 200 612 "-" "curl/7.65.1" "172.26.0.3"
bench_1    | Age: 0
bench_1    | Age: 1
bench_1    | Age: 2
bench_1    | Age: 3
nginx_1    | 172.26.0.6 - - [15/Sep/2019:16:27:01 +0000] "GET / HTTP/1.1" 200 612 "-" "curl/7.65.1" "172.26.0.3"
bench_1    | Age: 0
bench_1    | Age: 5
bench_1    | Age: 2
bench_1    | Age: 3
varnish_1  | 172.26.0.3 - - [15/Sep/2019:16:27:05 +0000] "PURGE / HTTP/1.1" 202 0 "" "curl/7.65.1"                      <-- http-broadcast receives a PURGE request
varnish_1  | 127.0.0.1 - - [15/Sep/2019:16:27:05 +0000] "PURGE / HTTP/1.1" 200 240 "-" "curl/7.65.1"                    <-- PURGE request received by varnishs instances
varnish_3  | 127.0.0.1 - - [15/Sep/2019:16:27:05 +0000] "PURGE / HTTP/1.1" 200 240 "-" "curl/7.65.1"                    <-- PURGE request received by varnishs instances
varnish_2  | 127.0.0.1 - - [15/Sep/2019:16:27:05 +0000] "PURGE / HTTP/1.1" 200 240 "-" "curl/7.65.1"                    <-- PURGE request received by varnishs instances
nginx_1    | 172.26.0.5 - - [15/Sep/2019:16:27:05 +0000] "GET / HTTP/1.1" 200 612 "-" "curl/7.65.1" "172.26.0.3"        <-- nginx is called because of cache missed of varnish
bench_1    | Age: 0                                                                                                     <-- bench receives fresh new response
nginx_1    | 172.26.0.7 - - [15/Sep/2019:16:27:06 +0000] "GET / HTTP/1.1" 200 612 "-" "curl/7.65.1" "172.26.0.3"
bench_1    | Age: 0
nginx_1    | 172.26.0.6 - - [15/Sep/2019:16:27:07 +0000] "GET / HTTP/1.1" 200 612 "-" "curl/7.65.1" "172.26.0.3"
bench_1    | Age: 0
bench_1    | Age: 1
bench_1    | Age: 4
```
