# gRPC TLS client

An example gRPC client supporting various TLS connection modes and other dial
options.

## Build

```
go build
```

## Run

```
./grpc-tls-client <grpc_address> <tls_mode> [<http_username> <http_password> [<keepalive_time>]]
```

`tls_mode`:
- `"off"`: Use non-encrypted connection
- `"off-alt"`: Use non-encrypted connection (alternative)
- `"insecure"`: Use TLS without certificate checks
- `"ca"`: Use TLS with custom CA certificate file
- `"system"`: Use TLS with system certificates and TLS v1.2
- `"system-alt"`: Use TLS with system certificates (alternative)
`username`, `password`: Use HTTP Basic authentication
`keepalive_time`: Send keep-alive requests
