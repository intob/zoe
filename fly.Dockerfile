## Thanks to https://github.com/chemidy/smallest-secured-golang-docker-image
FROM golang:alpine as builder

RUN apk update \
    && apk add --no-cache \
      git \
      ca-certificates \
      tzdata \
    && update-ca-certificates

ENV USER=appuser
ENV UID=10001
# See https://stackoverflow.com/a/55757473/12429735
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

WORKDIR $GOPATH/src/lstn/
COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' -a \
    -o /go/bin/lstn .

FROM scratch

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /go/bin/lstn /go/bin/lstn

# Use an unprivileged user
#USER appuser:appuser

ENTRYPOINT ["/go/bin/lstn"]
