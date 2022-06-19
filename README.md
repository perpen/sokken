# sokken - tcp tunnelling via websockets

The sokken client listens on local ports, pipes them to remote services via
sokken servers.

Say you want to ssh from machine A to machine B, but machine B can only expose
an HTTP port (or your only access to B is through an http proxy which supports
websockets).

Start client on machine A:
```
# The sokken client will be listening on port 2222 on the local interface,
# and connections to this port will be tunneled to port 22 of machine-b.
machine-a $ sokken client :1234 127.0.0.1:2222 ws://machine-b:8000/tunnel/127.0.0.1:22
INF tunnelling 127.0.0.1:2222 to ws://machine-b:8000/tunnel/127.0.0.1:22
...
```
Start server on machine B:
```
# The sokken server will be listening on port 8000, and will accept tunneling
# requests for port 22.
machine-b $ sokken server :8000 127.0.0.1:22
INF listening on :8000, tunnelling to: [127.0.0.1:22]
...
```
Then from machine A you can invoke an ssh session on machine B:
```
machine-a $ ssh -p 2222 127.0.0.1
machine-b $
```

The sokken server can expose multiple addresses. Usually these addresses
will be for services on its own host, but they can be external too (see
`github.com:443` example below).

The sokken client can tunnel multiple ports to one or more sokken servers.

Logging is structured json, to a file or stderr. File rotation is self-managed.

## Examples

```
# Usage
machine-a $ sokken
Usage: sokken server [OPTION]... API_ADDR REMOTE_ADDR [REMOTE_ADDR_2...]
       sokken client [OPTION]... API_ADDR LISTEN_ADDR REMOTE_ADDR [LISTEN_ADDR_2 REMOTE_ADDR_2 ...]
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
  -max-connections int
        when reached further connections are rejected (default 100)

# machine-b exposes 2 addresses: its own sshd, and github
machine-b $ sokken server :8000 \
  127.0.0.1:22 \
  github.com:443

# Port 2222 is only available from machine-a, since the sokken client
# is listening on local interface
# The client exposes a health endpoint on :1234
machine-a $ sokken client :1234 127.0.0.1:2222 ws://machine-b:8000/tunnel/127.0.0.1:22

# Port 2222 is available from other machines, since the sokken client
# is listening on all interfaces
machine-a $ sokken client :1234 :2222 ws://machine-b:8000/tunnel/127.0.0.1:22

# machine-b and machine-c run sokken servers, we can  access them both from
# a single sokken client on machine-a
machine-a $ sokken client :1234  \
  127.0.0.1:2222 ws://machine-b:8000/tunnel/127.0.0.1:22 \
  127.0.0.1:8443 ws://machine-b:8000/tunnel/github.com:443 \
  127.0.0.1:8080 ws://machine-c:8000/tunnel/127.0.0.1:8080
```

## Monitoring

Both server and client expose a health endpoint:
```
$ curl http://machine-a:1234/health # or curl http://machine-b:8000/health
{
  "connections": 2,
  "max-connections": 100,
  "connections-used-percent": 0.02
}
```

## Hacking

This code is mostly made of low-level network calls, so difficult to unit test.

While coding I used `./testing/tester` to run client/server/etc on changes, see
this script. It would be better to create a test suite using containers.

## FIXME

- custom ca for client
- handle panics with structured logging
- test suite in containers
- test through the d*p proxy
  - haproxy works with ./testing/haproxy.conf
- performance test:
  - nfs3
  - nfs3 over openvpn
- pass some info from client to server, to allow easier identification in log
