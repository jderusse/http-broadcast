# Cookbooks

## How to get a valid JWT token?

In order to Published and Subscribe to the mercure's hub you need a valid JWT token.
See `HUB_PUBLISH_TOKEN`, `HUB_SUBSCRIBE_TOKEN` and `HUB_TOKEN` from [configuration documentation](configuration.md)

This token should be signed with the `JWT_KEY` used in mercure's HUB ([example](https://jwt.io/#debugger-io?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJtZXJjdXJlIjp7InB1Ymxpc2giOltdLCJzdWJzY3JpYmUiOltdfX0.Pcdsjf9mQKhEen8QD43UlZnat2E3-glbYnjF1yXkoCM)).

## Embedded in your own Docker image

When using docker without K8s, embedding the `http-broacaster` inside the
target is the easiest solution to scale and configure the agent.
Each time a `varnish` instance is started, an agent, is ready to broadcast
messages to it.
Client just have to sent invalidation request to the `http-broadcast` by
addressing the request to the `varnish` container with the right port.

```Dockerfile
FROM alpine

# Grab binary from official Docker image
COPY --from=jderusse/http-broadcast@latest /http-broadcast /bin/http-broadcast

# listen to port 6082
ENV SERVER_ADDR=:6082

# then broadcast to port 6081 (listen by varnish below)
ENV AGENT_ENDPOINT=http://127.0.0.1:6081

# install your main service
RUN apk add --no-cache \
    varnish

# update the entrypoint to start both services
# see https://docs.docker.com/config/containers/multi-service_container/
ENTRYPOINT ["/entrypoint.sh"]
CMD ["-a :6081"]
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

```ini
[program:http-broadcast]
command=http-broadcast
autostart=true
autorestart=true
stopsignal=QUIT
stopwaitsecs=30
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0
```

## Configure varnish ACL

Now message is sent througth a not-exposed port (the port `6082` should not
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

## Example

See [other examples](../examples) in this repository.
