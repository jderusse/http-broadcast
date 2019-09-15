# HTTP Broadcast

A scalable and fault resilient HTTP Broadcaster built on top of [mercure](https://github.com/dunglas/mercure)
able to forward an HTTP request to several servers without need to maintain a
registry.

This project has been initialized in order to invalidate a cluster of docker
varnish servers.

## Description

Each HTTP request sent to a broadcaster is pushed into a mercure HUB,
then dispatched to all other broadcaster who'll replay the same request.

```
                                           +-----------+   +--------+
                              /-------\    |           |   |        |
                             +         +   |   HTTP    +---> Server |
             +-----------+   |\-------/|   | Broadcast |   |        |
+--------+   |           |   |         <---+           |   +--------+
|        |   |   HTTP    |   | Mercure |   +-----------+
| client +---> Broadcast +--->   HUB   |
|        |   |           |   |         |   +-----------+
+--------+   +-----------+   |         <---+           |   +--------+
                              \-------/    |   HTTP    |   |        |
                                           | Broadcast +---> Server |
                                           |           |   |        |
                                           +-----------+   +--------+
```

## Benefits over other solution

* No needs to maintain an index: each server is allowed to start and die
  without need to be registered somewhere
* Fault tolerant: if a server is temporary unreachable (network issue for
  instance), messages won't be lost: all missed messages will be re-play once
  recovered.
* Scalable: dealing with 1 or 2 000 servers is transparent for the client, it
  have a single request to perform.
* Network secured: Servers don't have to be exposed to the client which is
  complexe to setup when the servers are located in differents region of the
  world.

## Use cases

Example of usage: Send PURGE/BAN request to a cluster of Varnish servers.
The solution will be to embeded a http-broadcast binary in each Varnish
Server (One could use the Inter-container network communication provided by
Kubernets Pods). By sending a single request to the varnish containers, the
http-broadcast will take care to dispatch the same request to all varnish
instances.

## Usage

### Prebuilt Binary

Grab the binary corresponding to your operating system and architecture from
the release page, then run:

```bash
$ SERVER_ADDR=0.0.0.0:6801 AGENT_ENDPOINT=http://localhost:6800 HUB_TOKEN=<valid-jwt-token> HUB_ENDPOINT=https://example.com/hub http-broadcast
```
### Docker Container

A Docker image is available on Docker Hub. The following command is enough to
get a working server in demo mode:

```bash
docker run -d --name mercure-hub \
    -e JWT_KEY='!ChangeMe!' \
    dunglas/mercure

docker run --rm -ti \
    -e SERVER_ADDR=0.0.0.0:6801 \
    -e AGENT_ENDPOINT=http://varnish:6081 \
    -e HUB_TOKEN=<valid-jwt-token> HUB_ENDPOINT=http://mercure-hub/hub \
    -p 6801:6801
    jderusse/http-broadcast

curl http://localhost:6801/foo/bar -X PURGE
```

### Embeded in your own Docker image

When using docker without K8s, embedding the `http-broacaster` inside the
target is the easiest solution to scale and configure the agent.
Each time a `varnish` instance is started, an agent, is ready to broadcast
messages to it.
Client just have to sent invalidation request to the `http-broadcast` by
addressing the request to the `varnish` container with the right port.

```Dockerfile
FROM alpine

COPY --from=jderusse/http-broadcast@latest /http-broadcast /bin/http-broadcast
ENV SERVER_ADDR=:6083
ENV AGENT_ENDPOINT=http://127.0.0.1:6082

# install your main service
RUN apk add --no-cache \
    varnish

# update the entrypoint to start both services
# see https://docs.docker.com/config/containers/multi-service_container/
ENTRYPOINT ["/entrypoint.sh"]
CMD ["-a :6081 -a :6082"]
```

with a naive `entrypoint.sh`

```bash
#!/bin/bash
/bin/http-broadcast &
broadcastPid=$!

/usr/sbin/varnishd "$@" &
varnishPid=$!

while sleep 60; do
  kill -0 $broadcastPid || exit 1
  kill -0 $varnishPid || exit 1
done
```

one could use [supervisord](http://supervisord.org/) which is dedicated for such job.

Now message is sent througth a not-exposed port (the port `6083` should not
be publicly accessible expected from the one allowed to send requests).
securising varnish is done by

```vcl
acl local {
    "localhost";
}

sub vcl_recv {
  if (req.method == "PURGE") {
    if (client.ip ~ local) {
       return(purge);
    } else {
       return(synth(403, "Access denied."));
    }
  }
}
```

### Configuration

* `AGENT_ENDPOINT`: the address to broadcast requests to (example: `127.0.0.1:6800`). When not defined, the broadcaster will only listen on requests. `SERVER_ADDR` or `AGENT_ENDPOINT` is required.
* `AGENT_RETRY_DELAY`: maximum duration for retrying the replay of the request. default to `60s`.
* `DEBUG`: set to `1` to enable the debug mode (prints recovery stack traces).
* `HUB_ENDPOINT`: the address of the the mercure hub to push and fetch messages (example: `https://example.com/hub`).
* `HUB_GUARD_TOKEN`: the token used to prevent infinite loop (in case an agent broadcast request to iteself). default to value resolved for `HUB_TOPIC`.
* `HUB_PUBLISH_TOKEN`: valid JWT token to allow publishing (can be omited if `HUB_TOKEN` is set).
* `HUB_SUBSCRIBE_TOKEN`: valid JWT token to allow subscribing (can be omited if `HUB_TOKEN` is set).
* `HUB_TIMEOUT`: maximum duration for pushing the message into the HUB, set to `0s` to disable. default to `5s`.
* `HUB_TOKEN`: valid JWT token to allow both publishing and subscribing.
* `HUB_TOPIC`: name of the Mercur's topic to exchange messages. default to `http-broadcast`. (This parameter can also be define with by the queryString ot `HUB_ENDPOINT`. example `HUB_ENDPOINT=https://example.com/hub?topic=my_topic`)
* `LOG_FORMAT`: the log format, can be `json`, `fluentd` or `text` (default).
* `LOG_LEVEL`: the log verbosity, can be `trace`, `debug`, `info` (default), `warn`, `error`, `fatal`.
* `SERVER_ADDR`: the address to listen on (example: `0.0.0.0:6081`). When not defined, the broadcaster will only pusblish requests. `SERVER_ADDR` or `AGENT_ENDPOINT` is required.
* `SERVER_CORS_ALLOWED_ORIGINS`: a comma separated list of allowed CORS origins, can be `*` for all.
* `SERVER_INSECURE`: trust everyone in [ProxyProtocol](https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt). default to DEBUG
* `SERVER_READ_TIMEOUT`: maximum duration before timing out writes of the response, set to `0s` to disable (default), example: `2m`.
* `SERVER_TLS_ACME_ADDR`: the address use by the acme server to listen on (example:  `0.0.0.0:8080`). defaut to `:http`
* `SERVER_TLS_ACME_CERT_DIR`: the directory where to store Let's Encrypt certificates
* `SERVER_TLS_ACME_HOSTS`: a comma separated list of hosts for which Let's Encrypt certificates must be issued
* `SERVER_TLS_CERT_FILE`: a cert file (to use a custom certificate)
* `SERVER_TLS_KEY_FILE`: a key file (to use a custom certificate)
* `SERVER_TRUSTED_IPS`: list of trusted ips which lead to remote client address replacement in [ProxyProtocol](https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt).
* `SERVER_WRITE_TIMEOUT`: maximum duration for reading the entire request, including the body, set to `0s` to disable (default), example: `2m`.

## TODO
- example K8s
