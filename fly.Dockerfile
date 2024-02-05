# Thanks to https://github.com/chemidy/smallest-secured-golang-docker-image
#
# STEP 1 build executable binary
#
FROM golang:alpine as builder

RUN apk update \
    && apk add --no-cache \
      git \
      ca-certificates \
      tzdata \
    && update-ca-certificates

WORKDIR /app
COPY . /app

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' -a \
    -o /app/lstn .

#
# STEP 2 build small image
#
FROM scratch

# Copy zoneinfo for time zone support
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
# Copy zoneinfo for time zone support
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Copy binary and app files
COPY --from=builder /app/lstn /lstn
COPY --from=builder /app/client.js /client.js

ENTRYPOINT ["/lstn"]
