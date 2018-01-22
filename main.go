package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"regexp"

	"github.com/acls/sniproxy/config"
	"github.com/golang/glog"
)

func getServerName(buf []byte) string {
	// check if tls record type
	if recordType(buf[0]) == recordTypeHandshake {
		// tls
		return getSNIServerName(buf)
	}
	// not tls
	return getHost(buf)
}

func getSNIServerName(buf []byte) string {
	n := len(buf)
	if n < 5 {
		glog.Info("not tls handshake")
		return getHost(buf)
	}

	// tls major version
	if buf[1] != 3 {
		glog.Error("TLS version < 3 not supported")
		return ""
	}

	// payload length
	//l := int(buf[3])<<16 + int(buf[4])

	//log.Printf("length: %d, got: %d", l, n)

	// handshake message type
	if uint8(buf[5]) != typeClientHello {
		glog.Error("not client hello")
		return ""
	}

	// parse client hello message

	msg := &clientHelloMsg{}

	// client hello message not include tls header, 5 bytes
	ret := msg.unmarshal(buf[5:])
	if !ret {
		glog.Error("parse hello message return false")
		return ""
	}
	return msg.serverName
}

var hostRegx = regexp.MustCompile("\r\nHost: (.*)\r\n")

func getHost(buf []byte) string {
	matches := hostRegx.FindStringSubmatch(string(buf))
	if len(matches) < 1 {
		glog.Error("failed to find host")
		return ""
	}
	return matches[1]
}

func forward(c net.Conn, data []byte) {
	addr := c.LocalAddr().(*net.TCPAddr)

	src := getServerName(data)
	if src == "" {
		src = string(addr.IP)
	}

	dst := cfg.ForwardRules.Get(src, addr.Port)
	if dst == "" {
		dst = cfg.Default
		if dst == "" {
			glog.Errorf("No dst address for: %s", src)
			return
		}
	}
	glog.Infof("Forward: %s -> %s", src, dst)

	c1, err := net.Dial("tcp", dst)
	if err != nil {
		glog.Error(err)
		return
	}

	defer c1.Close()

	if _, err = c1.Write(data); err != nil {
		glog.Error(err)
		return
	}

	ch := make(chan struct{}, 2)

	go func() {
		io.Copy(c1, c)
		ch <- struct{}{}
	}()

	go func() {
		io.Copy(c, c1)
		ch <- struct{}{}
	}()

	<-ch
}

func serve(c net.Conn) {
	defer c.Close()

	buf := make([]byte, 1024)
	n, err := c.Read(buf)
	if err != nil {
		glog.Error(err)
		return
	}

	forward(c, buf[:n])
}

var cfg config.Config

func main() {
	var cfgfile string
	flag.StringVar(&cfgfile, "c", "config.yaml", "config file")
	flag.Set("logtostderr", "true")
	flag.Parse()

	if c, err := config.ReadConfigFile(cfgfile); err != nil {
		glog.Fatal(err)
	} else {
		cfg = *c
	}

	for _, d := range cfg.Listen {
		glog.Infof("listen on :%d", d)
		l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", d))
		if err != nil {
			glog.Fatal(err)
		}
		go func(l net.Listener) {
			defer l.Close()
			for {
				c1, err := l.Accept()
				if err != nil {
					glog.Fatal(err)
				}
				go serve(c1)
			}
		}(l)
	}
	select {}
}
