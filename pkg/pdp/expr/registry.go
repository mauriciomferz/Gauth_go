package expr

import (
	"fmt"
	"sync"
	"time"
)

// FunctionArg represents an argument value passed to a function
type FunctionArg struct {
	StringValue string
	NumericValue float64
	BoolValue   bool
	TimeValue   time.Time
	Type        ArgType
}

// ArgType indicates the type of a function argument
type ArgType int

const (
	ArgTypeString ArgType = iota
	ArgTypeNumeric
	ArgTypeBool
	ArgTypeTime
)

// FunctionResult represents the return value from a function
type FunctionResult struct {
	BoolValue    bool
	StringValue  string
	NumericValue float64
	TimeValue    time.Time
	Type         ResultType
}

// ResultType indicates the type of function result
type ResultType int

const (
	ResultTypeBool ResultType = iota
	ResultTypeString
	ResultTypeNumeric
	ResultTypeTime
)

// Function is the signature for registered ABAC functions
// Takes attributes, current time, and parsed arguments
// Returns a result or error
type Function func(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error)

// FunctionMetadata describes a registered function
type FunctionMetadata struct {
	Name        string
	Description string
	MinArgs     int
	MaxArgs     int
	ArgTypes    []ArgType
	ReturnType  ResultType
	Category    string // e.g., "string", "numeric", "time", "collection"
}

// FunctionRegistry manages registered ABAC expression functions
type FunctionRegistry struct {
	mu        sync.RWMutex
	functions map[string]*registeredFunction
	metrics   *RegistryMetrics
}

type registeredFunction struct {
	fn       Function
	metadata FunctionMetadata
}

// RegistryMetrics tracks function registry usage
type RegistryMetrics struct {
	mu                sync.RWMutex
	functionCalls     map[string]uint64
	functionErrors    map[string]uint64
	totalCalls        uint64
	totalErrors       uint64
	registrationCount uint64
}

// NewFunctionRegistry creates a new function registry with built-in functions
func NewFunctionRegistry() *FunctionRegistry {
	r := &FunctionRegistry{
		functions: make(map[string]*registeredFunction),
		metrics:   &RegistryMetrics{
			functionCalls:  make(map[string]uint64),
			functionErrors: make(map[string]uint64),
		},
	}
	
	// Register built-in functions
	r.registerBuiltIns()
	
	return r
}

// Register adds a custom function to the registry
func (r *FunctionRegistry) Register(metadata FunctionMetadata, fn Function) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Validate metadata
	if metadata.Name == "" {
		return fmt.Errorf("function name cannot be empty")
	}
	if metadata.MinArgs < 0 {
		return fmt.Errorf("MinArgs cannot be negative")
	}
	if metadata.MaxArgs < metadata.MinArgs && metadata.MaxArgs != -1 {
		return fmt.Errorf("MaxArgs must be >= MinArgs or -1 for unlimited")
	}
	
	// Check for duplicate
	if _, exists := r.functions[metadata.Name]; exists {
		return fmt.Errorf("function '%s' already registered", metadata.Name)
	}
	
	r.functions[metadata.Name] = &registeredFunction{
		fn:       fn,
		metadata: metadata,
	}
	
	r.metrics.mu.Lock()
	r.metrics.registrationCount++
	r.metrics.mu.Unlock()
	
	return nil
}

// Lookup retrieves a registered function by name
func (r *FunctionRegistry) Lookup(name string) (Function, FunctionMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	fn, exists := r.functions[name]
	if !exists {
		return nil, FunctionMetadata{}, fmt.Errorf("function '%s' not registered", name)
	}
	
	return fn.fn, fn.metadata, nil
}

// Call invokes a registered function with validation
func (r *FunctionRegistry) Call(name string, attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
	fn, metadata, err := r.Lookup(name)
	if err != nil {
		return FunctionResult{}, err
	}
	
	// Validate argument count
	if len(args) < metadata.MinArgs {
		return FunctionResult{}, fmt.Errorf("function '%s' requires at least %d arguments, got %d", name, metadata.MinArgs, len(args))
	}
	if metadata.MaxArgs != -1 && len(args) > metadata.MaxArgs {
		return FunctionResult{}, fmt.Errorf("function '%s' accepts at most %d arguments, got %d", name, metadata.MaxArgs, len(args))
	}
	
	// Validate argument types (if specified)
	if len(metadata.ArgTypes) > 0 {
		for i, expectedType := range metadata.ArgTypes {
			if i >= len(args) {
				break
			}
			if args[i].Type != expectedType {
				return FunctionResult{}, fmt.Errorf("function '%s' argument %d expects type %v, got %v", name, i, expectedType, args[i].Type)
			}
		}
	}
	
	// Record call
	r.metrics.mu.Lock()
	r.metrics.functionCalls[name]++
	r.metrics.totalCalls++
	r.metrics.mu.Unlock()
	
	// Invoke function
	result, err := fn(attrs, now, args)
	if err != nil {
		r.metrics.mu.Lock()
		r.metrics.functionErrors[name]++
		r.metrics.totalErrors++
		r.metrics.mu.Unlock()
		return FunctionResult{}, err
	}
	
	return result, nil
}

// List returns metadata for all registered functions
func (r *FunctionRegistry) List() []FunctionMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	list := make([]FunctionMetadata, 0, len(r.functions))
	for _, fn := range r.functions {
		list = append(list, fn.metadata)
	}
	return list
}

// ListByCategory returns functions in a specific category
func (r *FunctionRegistry) ListByCategory(category string) []FunctionMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var list []FunctionMetadata
	for _, fn := range r.functions {
		if fn.metadata.Category == category {
			list = append(list, fn.metadata)
		}
	}
	return list
}

// GetMetrics returns current registry metrics
func (r *FunctionRegistry) GetMetrics() map[string]interface{} {
	r.metrics.mu.RLock()
	defer r.metrics.mu.RUnlock()
	
	callsByFunc := make(map[string]uint64)
	errorsByFunc := make(map[string]uint64)
	
	for name, count := range r.metrics.functionCalls {
		callsByFunc[name] = count
	}
	for name, count := range r.metrics.functionErrors {
		errorsByFunc[name] = count
	}
	
	return map[string]interface{}{
		"total_calls":        r.metrics.totalCalls,
		"total_errors":       r.metrics.totalErrors,
		"registrations":      r.metrics.registrationCount,
		"function_calls":     callsByFunc,
		"function_errors":    errorsByFunc,
		"registered_count":   len(r.functions),
	}
}

// Unregister removes a function from the registry
func (r *FunctionRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.functions[name]; !exists {
		return fmt.Errorf("function '%s' not registered", name)
	}
	
	delete(r.functions, name)
	return nil
}

// Clear removes all registered functions (except built-ins if preserveBuiltIns is true)
func (r *FunctionRegistry) Clear(preserveBuiltIns bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if preserveBuiltIns {
		// Remove only custom functions
		for name, fn := range r.functions {
			if !isBuiltInCategory(fn.metadata.Category) {
				delete(r.functions, name)
			}
		}
	} else {
		r.functions = make(map[string]*registeredFunction)
	}
}

func isBuiltInCategory(category string) bool {
	builtInCategories := map[string]bool{
		"string":     true,
		"numeric":    true,
		"time":       true,
		"collection": true,
		"logical":    true,
		"comparison": true,
	}
	return builtInCategories[category]
}

// Helper functions to create function arguments
func StringArg(value string) FunctionArg {
	return FunctionArg{
		StringValue: value,
		Type:        ArgTypeString,
	}
}

func NumericArg(value float64) FunctionArg {
	return FunctionArg{
		NumericValue: value,
		Type:         ArgTypeNumeric,
	}
}

func BoolArg(value bool) FunctionArg {
	return FunctionArg{
		BoolValue: value,
		Type:      ArgTypeBool,
	}
}

func TimeArg(value time.Time) FunctionArg {
	return FunctionArg{
		TimeValue: value,
		Type:      ArgTypeTime,
	}
}

// Helper functions to create function results
func BoolResult(value bool) FunctionResult {
	return FunctionResult{
		BoolValue: value,
		Type:      ResultTypeBool,
	}
}

func StringResult(value string) FunctionResult {
	return FunctionResult{
		StringValue: value,
		Type:        ResultTypeString,
	}
}

func NumericResult(value float64) FunctionResult {
	return FunctionResult{
		NumericValue: value,
		Type:         ResultTypeNumeric,
	}
}

func TimeResult(value time.Time) FunctionResult {
	return FunctionResult{
		TimeValue: value,
		Type:      ResultTypeTime,
	}
}
