# Embed http-broacast in a Varnish image

This example shows how to configure a `http-broadcast` alongside a varnish
container in a Kubernetes pod to automatically broadcast the PURGE requests
to all the varnish containers.

## What's inside

- a pod containing a varnish and a `http-broadcast` container
- a dummy `nginx` pod
- a mercure hub pod

## How to demo

Run the containers and watch the logs

```bash
$ kubectl create namespace demo
$ kubectl -n demo apply -f app.yaml

$ kubectl -n demo logs -f deployment/varnish --all-containers=true --since=10m
$ kubectl -n demo logs -f deployment/bench --all-containers=true --since=10m
```

sample output

```bash
bench:   Age: 0                                                               <-- bench receives a 0 sec old response
bench:   Age: 0                                                               <-- bench receives a 0 sec old response from the second varnish pod
bench:   Age: 2
bench:   Age: 2
bench:   Age: 4
bench:   Age: 5
bench:   Age: 5
bench:   Age: 6
bench:   Age: 7
bench:   Age: 8
varnish: 192.168.140.113 - - "PURGE / HTTP/1.1" 202 0 "" "curl/7.65.1"        <-- http-broadcast replay the PURGE request
varnish: level=info msg="Agent: request played"
bench:   Age: 0                                                               <-- bench receives a 0 sec old response
bench:   Age: 1
bench:   Age: 0                                                               <-- bench receives a 0 sec old response from the second varnish pod
bench:   Age: 1
bench:   Age: 2
```
