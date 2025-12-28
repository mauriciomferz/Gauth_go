package expr

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// FunctionRegistry defines the interface for extensible function registration.
type FunctionRegistry interface {
	// Register adds a function to the registry.
	Register(name string, fn Function) error
	// Get retrieves a function by name.
	Get(name string) (Function, bool)
	// List returns all registered function names.
	List() []string
}

// Function represents a callable function in policy expressions.
// It takes a slice of arguments and returns a result or error.
type Function func(args []interface{}) (interface{}, error)

// DefaultRegistry is the global function registry with built-in functions.
var DefaultRegistry FunctionRegistry

func init() {
	DefaultRegistry = NewRegistry()
	// Register built-in functions
	_ = DefaultRegistry.Register("len", builtinLen)
	_ = DefaultRegistry.Register("upper", builtinUpper)
	_ = DefaultRegistry.Register("lower", builtinLower)
	_ = DefaultRegistry.Register("startsWith", builtinStartsWith)
	_ = DefaultRegistry.Register("endsWith", builtinEndsWith)
	_ = DefaultRegistry.Register("contains", builtinContains)
	_ = DefaultRegistry.Register("regex_match", builtinRegexMatch)
}

// regexCache internal cache for compiled expressions
var regexCache = struct {
	sync.RWMutex
	compiled map[string]*regexp.Regexp
	order    []string // simple LRU
	maxSize  int
}{
	compiled: make(map[string]*regexp.Regexp),
	order:    make([]string, 0, 100),
	maxSize:  100,
}

// registry is the concrete implementation of FunctionRegistry.
type registry struct {
	mu    sync.RWMutex
	funcs map[string]Function
}

// NewRegistry creates a new empty function registry.
func NewRegistry() FunctionRegistry {
	return &registry{
		funcs: make(map[string]Function),
	}
}

// Register adds a function to the registry.
func (r *registry) Register(name string, fn Function) error {
	if name == "" {
		return errors.New("function name cannot be empty")
	}
	if fn == nil {
		return errors.New("function cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.funcs[name]; exists {
		return fmt.Errorf("function %q already registered", name)
	}

	r.funcs[name] = fn
	return nil
}

// Get retrieves a function by name.
func (r *registry) Get(name string) (Function, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fn, ok := r.funcs[name]
	return fn, ok
}

// List returns all registered function names.
func (r *registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.funcs))
	for name := range r.funcs {
		names = append(names, name)
	}
	return names
}

// Built-in function implementations

// builtinLen returns the length of a string.
func builtinLen(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("len() requires exactly 1 argument, got %d", len(args))
	}

	str, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("len() argument must be a string, got %T", args[0])
	}

	return float64(len(str)), nil // Return as float64 for consistency with expression evaluator
}

// builtinUpper converts a string to uppercase.
func builtinUpper(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("upper() requires exactly 1 argument, got %d", len(args))
	}

	str, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("upper() argument must be a string, got %T", args[0])
	}

	return strings.ToUpper(str), nil
}

// builtinLower converts a string to lowercase.
func builtinLower(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("lower() requires exactly 1 argument, got %d", len(args))
	}

	str, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("lower() argument must be a string, got %T", args[0])
	}

	return strings.ToLower(str), nil
}

// builtinStartsWith checks if a string starts with a prefix.
func builtinStartsWith(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("startsWith() requires exactly 2 arguments, got %d", len(args))
	}

	str, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("startsWith() first argument must be a string, got %T", args[0])
	}

	prefix, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("startsWith() second argument must be a string, got %T", args[1])
	}

	return strings.HasPrefix(str, prefix), nil
}

// builtinEndsWith checks if a string ends with a suffix.
func builtinEndsWith(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("endsWith() requires exactly 2 arguments, got %d", len(args))
	}

	str, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("endsWith() first argument must be a string, got %T", args[0])
	}

	suffix, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("endsWith() second argument must be a string, got %T", args[1])
	}

	return strings.HasSuffix(str, suffix), nil
}

// builtinContains checks if a string contains a substring.
func builtinContains(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("contains() requires exactly 2 arguments, got %d", len(args))
	}

	str, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("contains() first argument must be a string, got %T", args[0])
	}

	substr, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("contains() second argument must be a string, got %T", args[1])
	}

	return strings.Contains(str, substr), nil
}

// builtinRegexMatch checks if a string matches a regular expression pattern.
func builtinRegexMatch(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("regex_match() requires exactly 2 arguments, got %d", len(args))
	}

	str, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("regex_match() first argument must be a string, got %T", args[0])
	}

	pattern, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("regex_match() second argument must be a string, got %T", args[1])
	}

	// Compile and match regex (with reasonable size limits)
	if len(pattern) > 256 {
		return nil, fmt.Errorf("regex_match() pattern too long (max 256 chars)")
	}

	// Check cache
	regexCache.RLock()
	re, ok := regexCache.compiled[pattern]
	regexCache.RUnlock()

	if !ok {
		// Compile and cache
		var err error
		re, err = regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("regex_match() invalid pattern: %w", err)
		}

		regexCache.Lock()
		// Double check
		if existing, exists := regexCache.compiled[pattern]; exists {
			re = existing
		} else {
			// Evict if full
			if len(regexCache.order) >= regexCache.maxSize {
				oldest := regexCache.order[0]
				delete(regexCache.compiled, oldest)
				regexCache.order = regexCache.order[1:]
			}
			regexCache.compiled[pattern] = re
			regexCache.order = append(regexCache.order, pattern)
		}
		regexCache.Unlock()
	}

	return re.MatchString(str), nil
}
