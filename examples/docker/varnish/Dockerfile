FROM alpine

COPY --from=jderusse/http-broadcast:latest /http-broadcast /bin/http-broadcast
ENV SERVER_ADDR=:6083
ENV AGENT_ENDPOINT=http://127.0.0.1:6082

# install your main service
RUN apk add --no-cache \
    varnish \
    supervisor

EXPOSE 6081 6082 6083
COPY ./files/. /

CMD ["supervisord", "-n", "-c", "/etc/supervisor/supervisord.conf", "-l", "/dev/stdout"]
