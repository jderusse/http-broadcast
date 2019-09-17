FROM alpine

COPY --from=jderusse/http-broadcast:latest /http-broadcast /bin/http-broadcast

# install your main service
RUN apk add --no-cache \
    varnish \
    supervisor

EXPOSE 6081 6082 6083
COPY ./files/. /

CMD ["supervisord", "-n", "-c", "/etc/supervisor/supervisord.conf", "-l", "/dev/stdout"]
