# Zoe
Zoe efficiently stores a very large number of tracking events.

Reports are generated periodically, and can be requested over http.

Built on only Gzip, Protobuf & Go's channels.

## Run locally
```bash
git rev-parse HEAD >> commit # commit id is served
go run .
```

## Deploy from scratch
### Launch app on Fly
```bash
fly launch
```
### Define secrets in GitHub repo
These are necessary for GitHub workflows.
```
AWS_PROD_ACCESS_KEY_ID
AWS_PROD_SECRET_ACCESS_KEY
AWS_HOSTED_ZONE_ID
FLY_API_TOKEN
```
### Run Deploy workflow
This will update the DNS records in the swissinfo.ch hosted zone & issue a TLS certificate on Fly.

## Maximum message size calculation
Adding the maximum sizes together:

EvType evType: 2 bytes
uint32 time: 6 bytes
fixed32 usr: 5 bytes
fixed32 sess: 5 bytes
uint32 cid: 6 bytes
optional uint32 pageSeconds: 6 bytes (if present)
optional float scrolled: 5 bytes (if present)

Total maximum size without optional fields: **24 bytes**
Total maximum size with all optional fields: **35 bytes**

To store on disk, we also need an additional byte as a length prefix.

Total maximum size including length prefix: **36 bytes**

## Why HTTP headers, no request body?
TLDR; it saves bandwidth & CPU cycles

HTTP headers themselves are not compressed by default in the HTTP/1.1 protocol. In HTTP/1.1, both request and response headers are sent as plain text. This means that if you send a request or a response with a lot of headers or cookies, the overhead can be quite significant, especially for small request/response bodies.

However, with the introduction of HTTP/2 and later HTTP/3, there are mechanisms to compress headers:

### HTTP/2

HTTP/2 introduced HPACK (Header Compression for HTTP/2), a specification for compressing headers. HPACK reduces the overhead of HTTP headers, making HTTP/2 more efficient than its predecessor, especially for use cases where many small requests and responses are made over the same connection. HPACK compresses headers before they are sent over the network and decompresses them on the other end.

### HTTP/3

HTTP/3, which builds on the QUIC transport protocol, uses QPACK for header compression. QPACK is designed to work well with the QUIC protocol's characteristics, providing header compression that allows for efficient, secure, and reliable transport over the internet. Like HPACK, QPACK compresses headers to reduce overhead but is specifically tailored to address some of the challenges presented by QUIC's design, such as ensuring that header compression does not negatively impact multiplexing or stream prioritization.

### Summary

While HTTP/1.1 does not compress headers, modern HTTP versions (HTTP/2 and HTTP/3) include mechanisms for header compression (HPACK and QPACK, respectively). These improvements significantly enhance the efficiency of web communication, especially for applications that make frequent or numerous HTTP requests.

## In progress

## To do

### Snapshots >> s3

## From David
- More meaningful & insightful analytics
- Which cids often group together in a single session?
