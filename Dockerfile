FROM alpine as extra

RUN apk add --no-cache \
        ca-certificates
#========================

FROM scratch

COPY http-broadcast /
COPY --from=extra /etc/ssl/certs/. /etc/ssl/certs/.
CMD ["/http-broadcast"]
EXPOSE 80 443
