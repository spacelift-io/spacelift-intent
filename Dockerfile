FROM alpine

RUN apk add --no-cache ca-certificates

COPY spacelift-intent /usr/local/bin/spacelift-intent
ENTRYPOINT ["/usr/local/bin/spacelift-intent"]
