package scripting

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/go-utils/id"
)

// Context represents the execution context for a Lua script
type Context struct {
	ID         string                 // Unique context ID
	HookName   string                 // Hook name
	Data       map[string]interface{} // Input/output data
	User       *UserContext           // Current user info
	Request    *HTTPContext           // HTTP request info
	Logger     *log.Helper            // Logger instance
	Cancel     context.Context        // Cancellation context
	Stopped    bool                   // Set to true if script calls stop()
	StopReason string                 // Reason for stopping
	StartTime  time.Time              // Execution start time
	mu         sync.RWMutex           // Protects Data map
}

// UserContext contains user information
type UserContext struct {
	ID       uint32   `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	TenantID uint32   `json:"tenant_id"`
}

// HTTPContext contains HTTP request information
type HTTPContext struct {
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	RemoteAddr string            `json:"remote_addr"`
	Headers    map[string]string `json:"headers"`
	Query      map[string]string `json:"query"`
}

// NewContext creates a new execution context
func NewContext(hookName string) *Context {
	return &Context{
		ID:        id.NewGUIDv4(false),
		HookName:  hookName,
		Data:      make(map[string]interface{}),
		StartTime: time.Now(),
	}
}

// Set sets a value in the context
func (c *Context) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Data[key] = value
}

// Get gets a value from the context
func (c *Context) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Data[key]
}

// GetString gets a string value from the context
func (c *Context) GetString(key string) string {
	if val := c.Get(key); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// GetInt gets an int value from the context
func (c *Context) GetInt(key string) int {
	if val := c.Get(key); val != nil {
		switch v := val.(type) {
		case int:
			return v
		case int32:
			return int(v)
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return 0
}

// GetBool gets a bool value from the context
func (c *Context) GetBool(key string) bool {
	if val := c.Get(key); val != nil {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// GetMap gets a map value from the context
func (c *Context) GetMap(key string) map[string]interface{} {
	if val := c.Get(key); val != nil {
		if m, ok := val.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}

// Has checks if a key exists in the context
func (c *Context) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.Data[key]
	return exists
}

// Delete removes a key from the context
func (c *Context) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Data, key)
}

// Clone creates a copy of the context
func (c *Context) Clone() *Context {
	c.mu.RLock()
	defer c.mu.RUnlock()

	newData := make(map[string]interface{}, len(c.Data))
	for k, v := range c.Data {
		newData[k] = v
	}

	return &Context{
		ID:         id.NewGUIDv4(false),
		HookName:   c.HookName,
		Data:       newData,
		User:       c.User,
		Request:    c.Request,
		Logger:     c.Logger,
		Cancel:     c.Cancel,
		Stopped:    c.Stopped,
		StopReason: c.StopReason,
		StartTime:  time.Now(),
	}
}

// WithUser sets the user context
func (c *Context) WithUser(user *UserContext) *Context {
	c.User = user
	return c
}

// WithRequest sets the HTTP request context
func (c *Context) WithRequest(req *HTTPContext) *Context {
	c.Request = req
	return c
}

// WithLogger sets the logger
func (c *Context) WithLogger(logger *log.Helper) *Context {
	c.Logger = logger
	return c
}

// WithContext sets the cancellation context
func (c *Context) WithContext(ctx context.Context) *Context {
	c.Cancel = ctx
	return c
}

// Stop stops further processing
func (c *Context) Stop(reason string) error {
	c.Stopped = true
	c.StopReason = reason
	if c.Logger != nil {
		c.Logger.Warnf("Context stopped: %s", reason)
	}
	return fmt.Errorf("execution stopped: %s", reason)
}

// Duration returns the elapsed time since context creation
func (c *Context) Duration() time.Duration {
	return time.Since(c.StartTime)
}

// ToMap converts context to a map for serialization
func (c *Context) ToMap() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	m := map[string]interface{}{
		"id":        c.ID,
		"hook_name": c.HookName,
		"data":      c.Data,
		"stopped":   c.Stopped,
		"duration":  c.Duration().String(),
	}

	if c.User != nil {
		m["user"] = c.User
	}

	if c.Request != nil {
		m["request"] = c.Request
	}

	if c.StopReason != "" {
		m["stop_reason"] = c.StopReason
	}

	return m
}
