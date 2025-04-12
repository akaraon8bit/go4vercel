package gee

import (
	"encoding/json"
	"fmt"
	"net/http"
	"context"
	"net"
	"strings"
)

type H map[string]interface{}

type Context struct {
	// origin objects
	Writer http.ResponseWriter
	Req    *http.Request
	// request info
	Path   string
	Method string
	Params map[string]string
	// response info
	StatusCode int
	// middleware
	handlers []HandlerFunc
	index    int
	// engine pointer
	engine *Engine
}

func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Path:   req.URL.Path,
		Method: req.Method,
		Req:    req,
		Writer: w,
		index:  -1,
	}
}

func (c *Context) Next() {
	c.index++
	s := len(c.handlers)
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}


func (c *Context) Header(key string) string {
	return c.Req.Header.Get(key)
}



// ClientIP implements a best effort algorithm to return the real client IP
func (c *Context) ClientIP() string {
	// Check for X-Forwarded-For header first
	if ip := c.Header("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}

	// Check for X-Real-Ip header
	if ip := c.Header("X-Real-Ip"); ip != "" {
		return ip
	}

	// Fall back to the remote address
	ip, _, err := net.SplitHostPort(c.Req.RemoteAddr)
	if err != nil {
		return c.Req.RemoteAddr
	}
	return ip
}


// QueryArray returns the query string values associated with the given key
func (c *Context) QueryArray(key string) []string {
	if values, ok := c.Req.URL.Query()[key]; ok {
		return values
	}
	return []string{}
}


// Abort stops the chain execution
func (c *Context) Abort() {
	c.index = len(c.handlers)
}

// AbortWithStatus stops the chain and sets the status code
func (c *Context) AbortWithStatus(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
	c.Abort()
}

func (c *Context) Fail(code int, err string) {
	c.index = len(c.handlers)
	c.JSON(code, H{"message": err})
}

func (c *Context) Param(key string) string {
	value, _ := c.Params[key]
	return value
}

func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}

func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

func (c *Context) SetHeader(key string, value string) {
	c.Writer.Header().Set(key, value)
}

func (c *Context) String(code int, format string, values ...interface{}) {
	c.SetHeader("Content-Type", "text/plain")
	c.Status(code)
	c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
}

func (c *Context) JSON(code int, obj interface{}) {
	c.SetHeader("Content-Type", "application/json")
	c.Status(code)
	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), 500)
	}
}

// ShouldBindJSON binds the request body to a struct
func (c *Context) ShouldBindJSON(obj interface{}) error {
	decoder := json.NewDecoder(c.Req.Body)
	return decoder.Decode(obj)
}

func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	c.Writer.Write(data)
}





// Add this to context.go
type contextKey struct {
    name string
}

func (c *Context) Set(key string, value interface{}) {
    if c.Req == nil {
        c.Req = &http.Request{}
    }
    if c.Req.Context() == nil {
        c.Req = c.Req.WithContext(context.Background())
    }
    ctx := context.WithValue(c.Req.Context(), contextKey{key}, value)
    c.Req = c.Req.WithContext(ctx)
}

func (c *Context) Get(key string) (interface{}, bool) {
    if c.Req == nil || c.Req.Context() == nil {
        return nil, false
    }
    value := c.Req.Context().Value(contextKey{key})
    return value, value != nil
}
