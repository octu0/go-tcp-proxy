# tcp-proxy

A small TCP proxy written in Go

This project was intended for debugging text-based protocols. The next version will address binary protocols.

## Usage

Since HTTP runs over TCP, we can also use `tcp-proxy` as a primitive HTTP proxy:

```
$ tcp-proxy -r httpstat.us:80
[  info ] 2019/07/31 19:47:36 main.go:66: lisen [0.0.0.0]:9999 proxy to httpstat.us:80
```

Then test with `curl`:

```
$ curl -H 'Host: httpstat.us' localhost:9999/418
418 I'm a teapot
```

### Match Example

```
$ tcp-proxy -r httpstat.us:80 --match 'Host: (.+)'
[  info ] 2019/07/31 19:55:37 main.go:48: starting up tcp-proxy/1.0.0
[  info ] 2019/07/31 19:55:37 main.go:66: lisen [0.0.0.0]:9999 proxy to httpstat.us:80
[  info ] 2019/07/31 19:55:37 main.go:109: matching Host: (.+)
[  info ] 2019/07/31 19:55:41 proxy.go:79: start proxy 127.0.0.1:58197 to 23.99.0.12:80
```

```
#run curl again...

[  info ] 2019/07/31 19:55:41 main.go:115: Matched #1: Host: httpstat.us
[  info ] 2019/07/31 19:55:42 proxy.go:87: close proxy 127.0.0.1:58197 to 23.99.0.12:80 (dur: 596.943946ms tx: 0.08 KB rx: 0.42 KB)
```

### Replace Example

```
$ tcp-proxy -r httpstat.us:80 --replace '(teapot)~*hello*'
[  info ] 2019/07/31 20:03:37 main.go:48: starting up tcp-proxy/1.0.0
[  info ] 2019/07/31 20:03:37 main.go:66: lisen [0.0.0.0]:9999 proxy to httpstat.us:80
[  info ] 2019/07/31 20:03:37 main.go:139: replacing (teapot) with *hello*
```

```
#run curl again...
$ curl -H 'Host: httpstat.us' localhost:9999/418
418 I'm a *hello
```

*Note: The `-replace` option is in the form `regex~replacer`. Where `replacer` may contain `$N` to substitute in group `N`.*

## Build

Build requires Go version 1.11+ installed.

```
$ go version
```

Run `make pkg` to Build and package for linux, darwin.

```
$ git clone https://github.com/octu0/go-tcp-proxy
$ make pkg
```

## Help

```
NAME:
   tcp-proxy

USAGE:
   tcp-proxy [global options] command [command options] [arguments...]

VERSION:
   1.0.0

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --log-dir value           /path/to/log directory (default: "/tmp") [$TCPPROXY_LOG_DIR]
   -l value, --local value   local address (default: "[0.0.0.0]:9999") [$TCPPROXY_LOCALADDR]
   -r value, --remote value  remote(destination) address (default: "127.0.0.1:8000") [$TCPPROXY_REMOTEADDR]
   -n, --no-nagles           disable nagles algorithm [$TCPPROXY_DISABLE_NAGLES]
   --unwrap-tls              remote connection with TLS exposed unencrypted locally [$TCPPROXY_UNWRAP_TLS]
   --match regex             match regex(in the form regex) [$TCPPROXY_MATCH]
   --replace regex-replacer  replace regex(in the form regex-replacer) [$TCPPROXY_REPLACE]
   -d, --debug               display server actions [$TCPPROXY_DEBUG]
   -V, --verbose             display server actions and all tcp data [$TCPPROXY_VERBOSE]
   --hex                     output binary data hexdecimal [$TCPPROXY_OUTPUT_HEX]
   --help, -h                show help
   --version, -v             print the version
```

*Note: Regex match and replace*
**only works on text strings**
*and does NOT work across packet boundaries*

### Todo

* Implement `tcpproxy.Conn` which provides accounting and hooks into the underlying `net.Conn`
* Verify wire protocols by providing `encoding.BinaryUnmarshaler` to a `tcpproxy.Conn`
* Modify wire protocols by also providing a map function
* Implement [SOCKS v5](https://www.ietf.org/rfc/rfc1928.txt) to allow for user-decided remote addresses
