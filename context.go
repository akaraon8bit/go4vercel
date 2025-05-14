package gee

import (
	"encoding/json"
	"fmt"
	"net/http"
	"context"
	"net"
	"strings"
	"strconv"
		"errors"
	"reflect"
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










// GetString returns the string value associated with the key from context
func (c *Context) GetString(key string) string {
	if val, ok := c.Get(key); ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// DefaultQuery returns the key's url query value if it exists, otherwise returns the defaultValue
func (c *Context) DefaultQuery(key, defaultValue string) string {
	if value := c.Query(key); value != "" {
		return value
	}
	return defaultValue
}

// StringToInt converts a string to int with error handling
func StringToInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// StringToInt64 converts a string to int64 with error handling
func StringToInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// GetInt returns the int value associated with the key from context
func (c *Context) GetInt(key string) int {
	if val, ok := c.Get(key); ok {
		if i, ok := val.(int); ok {
			return i
		}
	}
	return 0
}

// GetInt64 returns the int64 value associated with the key from context
func (c *Context) GetInt64(key string) int64 {
	if val, ok := c.Get(key); ok {
		if i, ok := val.(int64); ok {
			return i
		}
	}
	return 0
}

// GetFloat64 returns the float64 value associated with the key from context
func (c *Context) GetFloat64(key string) float64 {
	if val, ok := c.Get(key); ok {
		if f, ok := val.(float64); ok {
			return f
		}
	}
	return 0
}

// GetBool returns the bool value associated with the key from context
func (c *Context) GetBool(key string) bool {
	if val, ok := c.Get(key); ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// BindJSON is an alias for ShouldBindJSON for compatibility
func (c *Context) BindJSON(obj interface{}) error {
	return c.ShouldBindJSON(obj)
}

// HTML sends an HTTP response with content-type as text/html
func (c *Context) HTML(code int, html string) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	c.Writer.Write([]byte(html))
}

// ShouldBindQuery binds the query parameters to a struct
func (c *Context) ShouldBindQuery(obj interface{}) error {
	values := c.Req.URL.Query()
	return mapForm(obj, values)
}

// ... [keep all existing code below] ...

// Helper function for form binding
func mapForm(ptr interface{}, form map[string][]string) error {
	typ := reflect.TypeOf(ptr).Elem()
	val := reflect.ValueOf(ptr).Elem()

	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		structField := val.Field(i)
		if !structField.CanSet() {
			continue
		}

		inputFieldName := typeField.Tag.Get("form")
		if inputFieldName == "" {
			inputFieldName = typeField.Name
		}

		inputValue, exists := form[inputFieldName]
		if !exists {
			continue
		}

		numElems := len(inputValue)
		if structField.Kind() == reflect.Slice && numElems > 0 {
			sliceOf := structField.Type().Elem().Kind()
			slice := reflect.MakeSlice(structField.Type(), numElems, numElems)
			for i := 0; i < numElems; i++ {
				if err := setWithProperType(sliceOf, inputValue[i], slice.Index(i)); err != nil {
					return err
				}
			}
			val.Field(i).Set(slice)
		} else {
			if err := setWithProperType(typeField.Type.Kind(), inputValue[0], structField); err != nil {
				return err
			}
		}
	}
	return nil
}

// Helper function for setting values with proper type
func setWithProperType(valueKind reflect.Kind, val string, structField reflect.Value) error {
	switch valueKind {
	case reflect.Int:
		return setIntField(val, 0, structField)
	case reflect.Int8:
		return setIntField(val, 8, structField)
	case reflect.Int16:
		return setIntField(val, 16, structField)
	case reflect.Int32:
		return setIntField(val, 32, structField)
	case reflect.Int64:
		return setIntField(val, 64, structField)
	case reflect.Uint:
		return setUintField(val, 0, structField)
	case reflect.Uint8:
		return setUintField(val, 8, structField)
	case reflect.Uint16:
		return setUintField(val, 16, structField)
	case reflect.Uint32:
		return setUintField(val, 32, structField)
	case reflect.Uint64:
		return setUintField(val, 64, structField)
	case reflect.Bool:
		return setBoolField(val, structField)
	case reflect.Float32:
		return setFloatField(val, 32, structField)
	case reflect.Float64:
		return setFloatField(val, 64, structField)
	case reflect.String:
		structField.SetString(val)
	default:
		return errors.New("unknown type")
	}
	return nil
}

// Helper functions for setting specific types
func setIntField(val string, bitSize int, field reflect.Value) error {
	if val == "" {
		val = "0"
	}
	intVal, err := strconv.ParseInt(val, 10, bitSize)
	if err == nil {
		field.SetInt(intVal)
	}
	return err
}

func setUintField(val string, bitSize int, field reflect.Value) error {
	if val == "" {
		val = "0"
	}
	uintVal, err := strconv.ParseUint(val, 10, bitSize)
	if err == nil {
		field.SetUint(uintVal)
	}
	return err
}

func setBoolField(val string, field reflect.Value) error {
	if val == "" {
		val = "false"
	}
	boolVal, err := strconv.ParseBool(val)
	if err == nil {
		field.SetBool(boolVal)
	}
	return err
}

func setFloatField(val string, bitSize int, field reflect.Value) error {
	if val == "" {
		val = "0.0"
	}
	floatVal, err := strconv.ParseFloat(val, bitSize)
	if err == nil {
		field.SetFloat(floatVal)
	}
	return err
}
