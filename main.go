package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"regexp"
	"syscall"

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
		src = addr.IP.String()
	}

	config := cfg.Get()
	dst := config.ForwardRules.Get(src, addr.Port)
	if dst == "" {
		dst = config.Default
		if dst == "" {
			glog.Errorf("No dst address for ip:%s, src:%s", addr.IP.String(), src)
			return
		}
	}
	glog.Infof("Forward: %s:%d -> %s", src, addr.Port, dst)

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
		_, err := io.Copy(c1, c)
		if err != nil {
			glog.Error(err)
		}
		ch <- struct{}{}
	}()

	go func() {
		_, err := io.Copy(c, c1)
		if err != nil {
			glog.Error(err)
		}
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

var (
	cfgfile string
	cfg     config.ConfigLocker
)

func main() {
	var ctrlD bool
	flag.BoolVar(&ctrlD, "d", false, "handle ctrl+d")
	flag.StringVar(&cfgfile, "c", "config.yaml", "config file")
	flag.Set("logtostderr", "true")
	flag.Parse()

	if err := readConfig(); err != nil {
		glog.Fatal(err)
	}

	//@NOTE: listen ports are not reloaded with the rest of the config
	for _, d := range cfg.Get().Listen {
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

	handleSignals(ctrlD)

	select {} // don't exit
}

func handleSignals(ctrlD bool) {
	// Ctrl+C twice in a row before exiting
	// https://stackoverflow.com/a/18158859/467082
	listenForSignals(func() {
		// exit on first ctrl+c if ctrl+d isn't an option
		if !ctrlD {
			os.Exit(0)
		}

		ctrlCcount++
		if ctrlCcount < 2 {
			glog.Warning("Ctrl+C again to exit")
			return
		}
		glog.Warning("Ctrl+C pressed twice")
		os.Exit(0)
	}, os.Interrupt, syscall.SIGTERM)

	// reload config on SIGHUP
	listenForSignals(reloadConfig, syscall.SIGHUP)

	// Bad things happen if running as a systemd service and listening for ctrl+d.
	// e.g.: high cpu usage because of constant config reloading.
	if ctrlD {
		// reload config when ctrl+d is pressed
		// https://groups.google.com/forum/#!topic/Golang-Nuts/xeUTvBZsxp0
		// https://raw.githubusercontent.com/adonovan/gopl.io/master/ch1/dup1/main.go
	listenCtrlD:
		input := bufio.NewScanner(os.Stdin)
		for input.Scan() {
		}
		reloadConfig()
		goto listenCtrlD
	}
}

func listenForSignals(fn func(), sig ...os.Signal) {
	s := make(chan os.Signal, 1)
	signal.Notify(s, sig...)
	go func() {
		for {
			<-s
			fn()
		}
	}()
}

func readConfig() error {
	c, err := config.ReadConfigFile(cfgfile)
	if err != nil {
		return err
	}

	cfg.Set(c)
	return nil
}

var ctrlCcount = 0

func reloadConfig() {
	// reset Ctrl+C count
	ctrlCcount = 0

	if err := readConfig(); err != nil {
		glog.Warning(err)
		return
	}
	glog.Info("Config Reloaded")
}
