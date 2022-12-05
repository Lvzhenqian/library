package configs

import (
	"sync"
)

type Config[T any] interface {
	Reload(chan<- T)
}

type Module[T any] interface {
	Name() string
	Watch(<-chan T)
}

type ConfigManager[T any] struct {
	update chan T
	data   *T

	mux     *sync.RWMutex
	modules map[string]chan T
}

func NewManager[T any](cfg Config[T]) *ConfigManager[T] {
	update := make(chan T)
	go cfg.Reload(update)

	manager := &ConfigManager[T]{
		update:  update,
		mux:     new(sync.RWMutex),
		modules: make(map[string]chan T),
	}
	go manager.startNotify()

	return manager
}

func (c *ConfigManager[T]) AddModule(m Module[T]) {
	c.mux.Lock()
	defer c.mux.Unlock()
	// 新增一个 channel用来等待更新通知
	ch := make(chan T)
	c.modules[m.Name()] = ch
	// 开启一个协程来接收这个通知
	go m.Watch(ch)
}

func (c *ConfigManager[T]) RemoveModule(name string) {
	c.mux.Lock()
	defer c.mux.Unlock()
	channel, ok := c.modules[name]
	if ok {
		close(channel)
		delete(c.modules, name)
	}
}

func (c *ConfigManager[T]) startNotify() {
	for newData := range c.update {
		c.data = &newData

		c.mux.RLocker()
		for _, module := range c.modules {
			module <- newData
		}
		c.mux.RUnlock()
	}
}
