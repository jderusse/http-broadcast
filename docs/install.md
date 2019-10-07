# Install the HTTP Broadcaster

## Prebuilt Binary

Grab the binary corresponding to your operating system and architecture from
the release page, then run:

```bash
$ SERVER_ADDR=0.0.0.0:6801 AGENT_ENDPOINT=http://localhost:6800 HUB_TOKEN=<valid-jwt-token> HUB_ENDPOINT=https://example.com/hub http-broadcast
```
## Docker Container

A Docker image is available on Docker Hub. The following command is enough to
get a working server in demo mode:

```bash
docker run -d --name mercure-hub \
    -e JWT_KEY='!ChangeMe!' \
    dunglas/mercure

docker run --rm -ti \
    -e SERVER_ADDR=0.0.0.0:6801 \
    -e AGENT_ENDPOINT=http://varnish:6081 \
    -e HUB_TOKEN="<valid-jwt-token>" \
    -e HUB_ENDPOINT=http://mercure-hub/hub \
    -p 6801:6801
    jderusse/http-broadcast

curl http://localhost:6801/foo/bar -X PURGE
```

## Next step

[Configuration](configuration.md)
