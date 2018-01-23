Sniproxy
=======

A minimalistic SNI pass-through proxy implemented in golang. It doesn't do TLS termination or any load-balancing. It just routes connections by domain.

Routes HTTP and TLS connections:
- HTTP connections routed by hostname. The hostname is extracted from the HTTP "Host" header.
- TLS connections routed by SNI(Server Name Indication). The server name is extracted from the TLS ClientHello handshake.


# Getting started

## Install
```bash
go get github.com/acls/sniproxy
```

## Copy config
```bash
cp $GOPATH/src/github.com/acls/sniproxy/config.sample.yaml config.yaml
vim config.yaml
```

## Run
```bash
$GOPATH/bin/sniproxy -d -c config.yaml
```

# Example config

```yaml
# default destination
default: 127.0.0.1:8443
# listen on multiple ports
listen:
  - 80
  - 443
# forward rules - exact or wildcard matches
forward_rules:
  # forward by domain and port to 127.0.0.1:8080
  www.example.com:80: 127.0.0.1:8080
  # forward by domain to 127.0.0.1:8443
  www.example.com: 127.0.0.1:8443
  # wildcard match
  "*:80": "127.0.0.0:8080"
  # wildcard match and wildcard forward
  "*:9999": "*:443"
```

# As a Systemd Service

## Copy service file
NOTE: change ExecStart paths to match your paths, since the paths must be absolute. My $GOPATH is my home directory.

```bash
cp $GOPATH/src/github.com/acls/sniproxy/sniproxy.sample.service /etc/systemd/system/sniproxy.service
vim /etc/systemd/system/sniproxy.service
```

## Start service
```bash
systemctl start sniproxy.service
```

## View logs
```bash
journalctl -u sniproxy.service     # all logs
journalctl -u sniproxy.service -f  # follow logs
```

## Set to automatically run on boot
```bash
systemctl enable sniproxy.service
```

## Reload service without restarting after making changes to config
```bash
systemctl reload sniproxy.service
```

---
Based on https://github.com/fangdingjun/sniproxy
