package config

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/go-yaml/yaml"
)

type forwardRules map[string]string

func (fr forwardRules) Get(uri string, port int) (dst string) {
	var ok bool

	defer func() {
		if ok {
			// replace wildcard destination
			dst = strings.Replace(dst, "*", uri, 1)
		}
	}()

	// check domain and port
	if dst, ok = fr[fmt.Sprintf("%s:%d", uri, port)]; ok {
		return
	}
	// check domain
	if dst, ok = fr[uri]; ok {
		return
	}
	// check wildcard domain and port
	if dst, ok = fr[fmt.Sprintf("*:%d", port)]; ok {
		return
	}
	return
}

type Config struct {
	// Timeout      int
	Listen       []int
	Default      string
	ForwardRules forwardRules `yaml:"forward_rules"`
}

func ReadConfigFile(cfgfile string) (*Config, error) {
	var cfg Config

	data, err := ioutil.ReadFile(cfgfile)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
