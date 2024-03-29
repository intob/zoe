# Thanks to https://github.com/chemidy/smallest-secured-golang-docker-image
#
# STEP 1 build executable binary
#
FROM golang:alpine as builder

WORKDIR /app
COPY . /app

RUN apk update \
    && apk add --no-cache \
      git \
      ca-certificates \
      tzdata \
    && update-ca-certificates \
    && git rev-parse --short HEAD > commit \
    && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
      -ldflags='-w -s -extldflags "-static"' \
      -mod=readonly \
      -a \
      -o zoe .

#
# STEP 2 build small image
#
FROM scratch

# Copy zoneinfo for time zone support
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
# Copy binary and app files
COPY --from=builder /app/zoe /zoe
COPY --from=builder /app/assets /assets
COPY --from=builder /app/commit /commit

ENTRYPOINT ["/zoe"]
