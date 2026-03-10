package engine

import "sync"

// Context is a thread-safe key-value store shared across pipeline stages.
type Context struct {
	mu     sync.RWMutex
	values map[string]string
	logs   []string
}

func NewContext() *Context {
	return &Context{values: make(map[string]string)}
}

func (c *Context) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}

func (c *Context) Get(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.values[key]
}

func (c *Context) GetOr(key, fallback string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if v, ok := c.values[key]; ok {
		return v
	}
	return fallback
}

func (c *Context) AppendLog(entry string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logs = append(c.logs, entry)
}

func (c *Context) Snapshot() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]string, len(c.values))
	for k, v := range c.values {
		out[k] = v
	}
	return out
}

func (c *Context) Logs() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]string, len(c.logs))
	copy(out, c.logs)
	return out
}

func (c *Context) Clone() *Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	nc := NewContext()
	for k, v := range c.values {
		nc.values[k] = v
	}
	nc.logs = make([]string, len(c.logs))
	copy(nc.logs, c.logs)
	return nc
}

func (c *Context) ApplyUpdates(updates map[string]string) {
	if updates == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range updates {
		c.values[k] = v
	}
}
