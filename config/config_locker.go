package config

import (
	"sync/atomic"
	"unsafe"
)

type ConfigLocker struct {
	p unsafe.Pointer
}

func (c *ConfigLocker) Get() *Config {
	return (*Config)(atomic.LoadPointer(&c.p))
}
func (c *ConfigLocker) Set(newConfig *Config) {
	atomic.StorePointer(&c.p, unsafe.Pointer(newConfig))
}
