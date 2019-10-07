FROM alpine as extra

RUN apk add --no-cache \
        ca-certificates
#========================

FROM scratch

COPY --from=extra /etc/ssl/certs/. /etc/ssl/certs/.
CMD ["/http-broadcast"]
EXPOSE 80 443

COPY http-broadcast /
