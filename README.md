# sokken - tcp tunnelling via websockets

The sokken client listens on local ports, pipes them to remote services via
sokken servers.

Say you want to ssh from machine A to machine B, but machine B can only expose
an HTTP port (or your only access to B is through an http proxy which supports
websockets).

Start client on machine A:
```
machine-a $ sokken client 127.0.0.1:2222 ws://machine-b:8000/tunnel/127.0.0.1:22
INF tunnelling 127.0.0.1:2222 to ws://machine-b:8000/tunnel/127.0.0.1:22
...
```
Start server on machine B:
```
machine-b $ sokken server :8000 127.0.0.1:22
INF listening on :8000, tunnelling to: [127.0.0.1:22]
...
```
Then from machine A you can invoke an ssh session on machine B:
```
machine-a $ ssh -p 2222 127.0.0.1
machine-b $
```

The sokken server can expose multiple addresses.

The sokken client can forward multiple ports to one or more sokken servers.

Logging is structure json, to a file or stderr. File rotation is self-managed.

## Examples

```
# Usage
machine-a $ sokken
Usage: sokken server [OPTION]... LISTEN_ADDR REMOTE_ADDR [REMOTE_ADDR_2...]
       sokken client [OPTION]... LISTEN_ADDR REMOTE_ADDR [LISTEN_ADDR_2 REMOTE_ADDR_2 ...]
Options:
  -log-debug
        sets log level to DEBUG rather than INFO
  -log-file string
        sets path of log file, if absent log to stderr
  -log-max-age-days int
        max file age before rotation (default 1)
  -log-max-rotated int
        number of rotated files to keep (default 7)
  -log-max-size-mb int
        max file size before rotation (default 10)
  -log-pretty
        logs to console, in colourful non-json format - overrides log-file option

# machine-b exposes 2 addresses: its own sshd, and github
machine-b $ sokken server :8000 \
  127.0.0.1:22 \
  github.com:443

# Port 2222 is only available from machine-a, since the sokken client
# is listening on local interface
machine-a $ sokken client 127.0.0.1:2222 ws://machine-b:8000/tunnel/127.0.0.1:22

# Port 2222 is available from other machines, since the sokken client
# is listening on all interfaces
machine-a $ sokken client :2222 ws://machine-b:8000/tunnel/127.0.0.1:22

# machine-b and machine-c run sokken servers, we can  access them both from
# a single sokken client on machine-a
machine-a $ sokken client \
  127.0.0.1:2222 ws://machine-b:8000/tunnel/127.0.0.1:22 \
  127.0.0.1:8443 ws://machine-b:8000/tunnel/github.com:443 \
  127.0.0.1:8080 ws://machine-c:8000/tunnel/127.0.0.1:8080
```

## Monitoring of server

```
$ curl http://machine-b:8000/health
{
  "connections": 2,
  "max-connections": 100,
  "connections-capacity-percent": 2
}
```

# FIXME

- handle panics with structured logging
- tls, ca for client
- test through the d*p proxy, and our haproxy
- log 'x-forwarded-for' ?
- access logging?
- for ease of use in other contexts, optionally proxy all other requests to
  another server. What real use cases?
