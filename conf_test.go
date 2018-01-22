package main

import (
	"fmt"
	"testing"
)

func Test_forwardRules_Get(t *testing.T) {
	c, err := ReadConfigFile("config.sample.yaml")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%+v\n", c)
	var testdata = map[string]string{
		"www.example.com": "127.0.0.1:8443",
		"b.example.com":   "127.0.0.1:8541",
	}
	fr := forwardRules(c.ForwardRules)
	for k, v := range testdata {
		s, _ := fr[k]
		if s != v {
			t.Errorf("expected: %s, got: %s", v, s)
		}
	}

	type args struct {
		uri  string
		port int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"1", args{"www.example.com", 80}, "127.0.0.1:8080"},
		{"2", args{"www.example.com", 443}, "127.0.0.1:8443"},
		{"3", args{"www.example.com", 9999}, "127.0.0.1:8443"},
		{"4", args{"a.com", 9999}, "a.com:443"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fr.Get(tt.args.uri, tt.args.port); got != tt.want {
				t.Errorf("forwardRules.Get() got %v, want %v", got, tt.want)
			}
		})
	}
}
