package gapless

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"syscall"
)

// DictObj is our core datastore for the settings.
type DictObj struct {
	data     map[string]interface{}
	ConfFile string
}

// NewSettingsObj is just a convenience method for a new settings obj.
func NewSettingsObj() *DictObj {
	return &DictObj{data: make(map[string]interface{})}
}

// LoadFromFile takes a file path and populates the setting obj.
func (s *DictObj) LoadFromFile(filepath string) {
	f, err := os.Open(filepath)
	if err != nil {
		panic(fmt.Sprintf("Opening settings file failed: %s", err))
	}
	defer f.Close()

	b := new(bytes.Buffer)
	_, err = b.ReadFrom(f)
	if err != nil {
		panic(fmt.Sprintf("Reading settings file failed: %s", err))
	}

	err = json.Unmarshal(b.Bytes(), &s.data)
	if err != nil {
		panic(fmt.Sprintf("Unpacking settings json failed: %s", err))
	}

	s.ConfFile = filepath
}

// Set manually sets a single key / value pair.
func (s *DictObj) Set(key string, val interface{}) {
	s.data[key] = val
}

// SetFromEnv takes an environment variable name and sets that value to a
// specified key.
func (s *DictObj) SetFromEnv(key, envKey string, args ...interface{}) {
	envVal, ok := syscall.Getenv(envKey)

	if ok {
		s.data[key] = envVal
	}

	switch len(args) {
	case 0:
		break
	case 1:
		s.data[key] = args[0]
	default:
		panic(fmt.Sprintf("SetFromEnv received too many args: [%d]", len(args)))
	}
}

// Bool returns a bool settings value.
func (s *DictObj) Bool(key string, args ...bool) bool {
	def := false

	switch len(args) {
	case 0:
		break
	case 1:
		def = bool(args[0])
	default:
		panic(fmt.Sprintf("Bool received too many args: [%d]", len(args)))
	}

	x, ok := s.data[key]
	if !ok {
		return def
	}
	return x.(bool)
}

// Int returns an int settings value.
func (s *DictObj) Int(key string, args ...int) int {
	var def int = -1

	switch len(args) {
	case 0:
		break
	case 1:
		def = args[0]
	default:
		panic(fmt.Sprintf("Int received too many args: [%d]", len(args)))
	}

	x, ok := s.data[key]
	if !ok {
		return def
	}

	// Json will think an int is a float... check and cast please.
	if reflect.TypeOf(x).Kind() == reflect.Float64 {
		return int(x.(float64))
	}

	return x.(int)
}

// Float returns a float settings value.
func (s *DictObj) Float(key string, args ...float64) float64 {
	var def float64 = -1

	switch len(args) {
	case 0:
		break
	case 1:
		def = args[0]
	default:
		panic(fmt.Sprintf("Float received too many args: [%d]", len(args)))
	}

	x, ok := s.data[key]
	if !ok {
		return def
	}
	return x.(float64)
}

// String returns a string settings value.
func (s *DictObj) String(key string, args ...string) string {
	var def string

	switch len(args) {
	case 0:
		break
	case 1:
		def = args[0]
	default:
		panic(fmt.Sprintf("String received too many args: [%d]", len(args)))
	}

	result, present := s.data[key]
	if !present {
		return def
	}
	return result.(string)
}
