package expr

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// registerBuiltIns registers all built-in ABAC functions
func (r *FunctionRegistry) registerBuiltIns() {
	// String functions
	r.Register(FunctionMetadata{
		Name:        "contains",
		Description: "Check if string contains substring",
		MinArgs:     2,
		MaxArgs:     2,
		ArgTypes:    []ArgType{ArgTypeString, ArgTypeString},
		ReturnType:  ResultTypeBool,
		Category:    "string",
	}, fnContains)
	
	r.Register(FunctionMetadata{
		Name:        "starts_with",
		Description: "Check if string starts with prefix",
		MinArgs:     2,
		MaxArgs:     2,
		ArgTypes:    []ArgType{ArgTypeString, ArgTypeString},
		ReturnType:  ResultTypeBool,
		Category:    "string",
	}, fnStartsWith)
	
	r.Register(FunctionMetadata{
		Name:        "ends_with",
		Description: "Check if string ends with suffix",
		MinArgs:     2,
		MaxArgs:     2,
		ArgTypes:    []ArgType{ArgTypeString, ArgTypeString},
		ReturnType:  ResultTypeBool,
		Category:    "string",
	}, fnEndsWith)
	
	r.Register(FunctionMetadata{
		Name:        "regex_match",
		Description: "Match string against regular expression",
		MinArgs:     2,
		MaxArgs:     2,
		ArgTypes:    []ArgType{ArgTypeString, ArgTypeString},
		ReturnType:  ResultTypeBool,
		Category:    "string",
	}, fnRegexMatch)
	
	r.Register(FunctionMetadata{
		Name:        "to_upper",
		Description: "Convert string to uppercase",
		MinArgs:     1,
		MaxArgs:     1,
		ArgTypes:    []ArgType{ArgTypeString},
		ReturnType:  ResultTypeString,
		Category:    "string",
	}, fnToUpper)
	
	r.Register(FunctionMetadata{
		Name:        "to_lower",
		Description: "Convert string to lowercase",
		MinArgs:     1,
		MaxArgs:     1,
		ArgTypes:    []ArgType{ArgTypeString},
		ReturnType:  ResultTypeString,
		Category:    "string",
	}, fnToLower)
	
	r.Register(FunctionMetadata{
		Name:        "trim",
		Description: "Remove leading/trailing whitespace",
		MinArgs:     1,
		MaxArgs:     1,
		ArgTypes:    []ArgType{ArgTypeString},
		ReturnType:  ResultTypeString,
		Category:    "string",
	}, fnTrim)
	
	r.Register(FunctionMetadata{
		Name:        "str_length",
		Description: "Get string length",
		MinArgs:     1,
		MaxArgs:     1,
		ArgTypes:    []ArgType{ArgTypeString},
		ReturnType:  ResultTypeNumeric,
		Category:    "string",
	}, fnStrLength)
	
	// Numeric functions
	r.Register(FunctionMetadata{
		Name:        "abs",
		Description: "Absolute value",
		MinArgs:     1,
		MaxArgs:     1,
		ArgTypes:    []ArgType{ArgTypeNumeric},
		ReturnType:  ResultTypeNumeric,
		Category:    "numeric",
	}, fnAbs)
	
	r.Register(FunctionMetadata{
		Name:        "min",
		Description: "Minimum of two numbers",
		MinArgs:     2,
		MaxArgs:     2,
		ArgTypes:    []ArgType{ArgTypeNumeric, ArgTypeNumeric},
		ReturnType:  ResultTypeNumeric,
		Category:    "numeric",
	}, fnMin)
	
	r.Register(FunctionMetadata{
		Name:        "max",
		Description: "Maximum of two numbers",
		MinArgs:     2,
		MaxArgs:     2,
		ArgTypes:    []ArgType{ArgTypeNumeric, ArgTypeNumeric},
		ReturnType:  ResultTypeNumeric,
		Category:    "numeric",
	}, fnMax)
	
	// Time functions
	r.Register(FunctionMetadata{
		Name:        "time_between",
		Description: "Check if current time is between start and end (HH:MM format)",
		MinArgs:     2,
		MaxArgs:     2,
		ArgTypes:    []ArgType{ArgTypeString, ArgTypeString},
		ReturnType:  ResultTypeBool,
		Category:    "time",
	}, fnTimeBetween)
	
	r.Register(FunctionMetadata{
		Name:        "weekday",
		Description: "Get current weekday (0=Sunday, 6=Saturday)",
		MinArgs:     0,
		MaxArgs:     0,
		ReturnType:  ResultTypeNumeric,
		Category:    "time",
	}, fnWeekday)
	
	r.Register(FunctionMetadata{
		Name:        "is_weekend",
		Description: "Check if current time is weekend",
		MinArgs:     0,
		MaxArgs:     0,
		ReturnType:  ResultTypeBool,
		Category:    "time",
	}, fnIsWeekend)
	
	r.Register(FunctionMetadata{
		Name:        "hour",
		Description: "Get current hour (0-23)",
		MinArgs:     0,
		MaxArgs:     0,
		ReturnType:  ResultTypeNumeric,
		Category:    "time",
	}, fnHour)
	
	// Collection functions
	r.Register(FunctionMetadata{
		Name:        "in",
		Description: "Check if value is in list",
		MinArgs:     2,
		MaxArgs:     -1, // unlimited
		ArgTypes:    []ArgType{ArgTypeString}, // first arg checked, rest are list items
		ReturnType:  ResultTypeBool,
		Category:    "collection",
	}, fnIn)
	
	r.Register(FunctionMetadata{
		Name:        "not_in",
		Description: "Check if value is not in list",
		MinArgs:     2,
		MaxArgs:     -1,
		ArgTypes:    []ArgType{ArgTypeString},
		ReturnType:  ResultTypeBool,
		Category:    "collection",
	}, fnNotIn)
	
	// Logical functions
	r.Register(FunctionMetadata{
		Name:        "not",
		Description: "Logical NOT",
		MinArgs:     1,
		MaxArgs:     1,
		ArgTypes:    []ArgType{ArgTypeBool},
		ReturnType:  ResultTypeBool,
		Category:    "logical",
	}, fnNot)
}

// String function implementations

func fnContains(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
	haystack := args[0].StringValue
	needle := args[1].StringValue
	return BoolResult(strings.Contains(haystack, needle)), nil
}

func fnStartsWith(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
	str := args[0].StringValue
	prefix := args[1].StringValue
	return BoolResult(strings.HasPrefix(str, prefix)), nil
}

func fnEndsWith(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
	str := args[0].StringValue
	suffix := args[1].StringValue
	return BoolResult(strings.HasSuffix(str, suffix)), nil
}

type cachedRegexMap struct {
	sync.RWMutex
	m map[string]*regexp.Regexp
}

var registryRegexCache = &cachedRegexMap{m: make(map[string]*regexp.Regexp)}

func fnRegexMatch(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
	str := args[0].StringValue
	pattern := args[1].StringValue
	
	// Security: limit pattern length
	if len(pattern) > 256 {
		return BoolResult(false), fmt.Errorf("regex pattern too long (max 256 chars)")
	}
	
	registryRegexCache.RLock()
	re, cached := registryRegexCache.m[pattern]
	registryRegexCache.RUnlock()
	
	if !cached {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return BoolResult(false), fmt.Errorf("invalid regex pattern: %v", err)
		}
		
		registryRegexCache.Lock()
		// Check cache size to prevent unbounded growth
		if len(registryRegexCache.m) > 100 {
			// Simple eviction: clear cache
			registryRegexCache.m = make(map[string]*regexp.Regexp)
		}
		registryRegexCache.m[pattern] = compiled
		registryRegexCache.Unlock()
		
		re = compiled
	}
	
	return BoolResult(re.MatchString(str)), nil
}

func fnToUpper(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
	return StringResult(strings.ToUpper(args[0].StringValue)), nil
}

func fnToLower(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
	return StringResult(strings.ToLower(args[0].StringValue)), nil
}

func fnTrim(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
	return StringResult(strings.TrimSpace(args[0].StringValue)), nil
}

func fnStrLength(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
	return NumericResult(float64(len(args[0].StringValue))), nil
}

// Numeric function implementations

func fnAbs(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
	val := args[0].NumericValue
	if val < 0 {
		return NumericResult(-val), nil
	}
	return NumericResult(val), nil
}

func fnMin(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
	a := args[0].NumericValue
	b := args[1].NumericValue
	if a < b {
		return NumericResult(a), nil
	}
	return NumericResult(b), nil
}

func fnMax(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
	a := args[0].NumericValue
	b := args[1].NumericValue
	if a > b {
		return NumericResult(a), nil
	}
	return NumericResult(b), nil
}

// Time function implementations

func fnTimeBetween(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
	startStr := args[0].StringValue
	endStr := args[1].StringValue
	
	layout := "15:04"
	start, err := time.Parse(layout, startStr)
	if err != nil {
		return BoolResult(false), fmt.Errorf("invalid start time format (use HH:MM): %v", err)
	}
	
	end, err := time.Parse(layout, endStr)
	if err != nil {
		return BoolResult(false), fmt.Errorf("invalid end time format (use HH:MM): %v", err)
	}
	
	cur := now.UTC()
	curClock := time.Date(0, 1, 1, cur.Hour(), cur.Minute(), 0, 0, time.UTC)
	sClock := time.Date(0, 1, 1, start.Hour(), start.Minute(), 0, 0, time.UTC)
	eClock := time.Date(0, 1, 1, end.Hour(), end.Minute(), 0, 0, time.UTC)
	
	var result bool
	if sClock.Before(eClock) {
		// Normal range (e.g., 09:00-17:00)
		result = (curClock.Equal(sClock) || curClock.After(sClock)) && (curClock.Before(eClock) || curClock.Equal(eClock))
	} else {
		// Overnight range (e.g., 22:00-06:00)
		result = !(curClock.After(eClock) && curClock.Before(sClock))
	}
	
	return BoolResult(result), nil
}

func fnWeekday(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
	return NumericResult(float64(now.Weekday())), nil
}

func fnIsWeekend(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
	wd := now.Weekday()
	return BoolResult(wd == time.Saturday || wd == time.Sunday), nil
}

func fnHour(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
	return NumericResult(float64(now.Hour())), nil
}

// Collection function implementations

func fnIn(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
	value := args[0].StringValue
	for i := 1; i < len(args); i++ {
		if value == args[i].StringValue {
			return BoolResult(true), nil
		}
	}
	return BoolResult(false), nil
}

func fnNotIn(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
	value := args[0].StringValue
	for i := 1; i < len(args); i++ {
		if value == args[i].StringValue {
			return BoolResult(false), nil
		}
	}
	return BoolResult(true), nil
}

// Logical function implementations

func fnNot(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
	return BoolResult(!args[0].BoolValue), nil
}

// Attribute accessor helper for registry-based evaluation
func GetAttrValue(attrs map[string]string, key string) (FunctionArg, error) {
	val, exists := attrs[key]
	if !exists {
		return FunctionArg{}, fmt.Errorf("attribute '%s' not found", key)
	}
	
	// Try parsing as numeric
	if numVal, err := strconv.ParseFloat(val, 64); err == nil {
		return NumericArg(numVal), nil
	}
	
	// Try parsing as bool
	if boolVal, err := strconv.ParseBool(val); err == nil {
		return BoolArg(boolVal), nil
	}
	
	// Default to string
	return StringArg(val), nil
}
