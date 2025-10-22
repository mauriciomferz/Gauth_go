package expr

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Eval evaluates a boolean expression against attributes and current time.
// Returns false with error if parsing/evaluation fails.
func Eval(expression string, attrs map[string]string, now time.Time) (bool, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return true, nil
	}
	return parseOr(expression, attrs, now)
}

func parseOr(s string, attrs map[string]string, now time.Time) (bool, error) {
	parts := splitTopWithParens(s, "||")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := parseAnd(p, attrs, now)
		if err != nil {
			return false, err
		}
		if v {
			return true, nil
		}
	}
	return false, nil
}

func parseAnd(s string, attrs map[string]string, now time.Time) (bool, error) {
	parts := splitTopWithParens(s, "&&")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := parseUnary(p, attrs, now)
		if err != nil {
			return false, err
		}
		if !v {
			return false, nil
		}
	}
	return true, nil
}

func parseUnary(s string, attrs map[string]string, now time.Time) (bool, error) {
	neg := 0
	for strings.HasPrefix(strings.TrimSpace(s), "!") {
		neg++
		s = strings.TrimSpace(s)[1:]
	}
	v, err := parsePrimary(strings.TrimSpace(s), attrs, now)
	if err != nil {
		return false, err
	}
	if neg%2 == 1 {
		return !v, nil
	}
	return v, nil
}

func parsePrimary(s string, attrs map[string]string, now time.Time) (bool, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return true, nil
	}
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") && matchingParens(s) {
		inner := strings.TrimSpace(s[1 : len(s)-1])
		return parseOr(inner, attrs, now)
	}
	return evalClause(s, attrs, now)
}

var regexCache = struct {
	// very small cache; production would use LRU with eviction metrics
	compiled map[string]*regexp.Regexp
}{compiled: map[string]*regexp.Regexp{}}

func evalClause(clause string, attrs map[string]string, now time.Time) (bool, error) {
	if strings.HasPrefix(clause, "time_between") {
		i := strings.Index(clause, "(")
		j := strings.LastIndex(clause, ")")
		if i < 0 || j < 0 {
			return false, fmt.Errorf("invalid time_between syntax")
		}
		inside := clause[i+1 : j]
		segs := splitCSV(inside)
		if len(segs) != 2 {
			return false, fmt.Errorf("time_between requires 2 params")
		}
		layout := "15:04"
		start, err := time.Parse(layout, trimQuotes(segs[0]))
		if err != nil {
			return false, err
		}
		end, err := time.Parse(layout, trimQuotes(segs[1]))
		if err != nil {
			return false, err
		}
		cur := now.UTC()
		curClock := time.Date(0, 1, 1, cur.Hour(), cur.Minute(), 0, 0, time.UTC)
		sClock := time.Date(0, 1, 1, start.Hour(), start.Minute(), 0, 0, time.UTC)
		eClock := time.Date(0, 1, 1, end.Hour(), end.Minute(), 0, 0, time.UTC)
		if sClock.Before(eClock) {
			return (curClock.Equal(sClock) || curClock.After(sClock)) && (curClock.Before(eClock) || curClock.Equal(eClock)), nil
		}
		return !(curClock.After(eClock) && curClock.Before(sClock)), nil // overnight window
	}
	if strings.Contains(clause, " in ") {
		segs := strings.SplitN(clause, " in ", 2)
		key := strings.TrimSpace(segs[0])
		list := strings.TrimSpace(segs[1])
		if !strings.HasPrefix(list, "[") || !strings.HasSuffix(list, "]") {
			return false, fmt.Errorf("invalid in list syntax")
		}
		list = strings.Trim(list, "[]")
		opts := splitCSV(list)
		val := attrs[key]
		for _, o := range opts {
			if val == trimQuotes(o) {
				return true, nil
			}
		}
		return false, nil
	}
	if strings.Contains(clause, "==") {
		segs := strings.SplitN(clause, "==", 2)
		key := strings.TrimSpace(segs[0])
		want := trimQuotes(strings.TrimSpace(segs[1]))
		return attrs[key] == want, nil
	}
	// contains(key,'substr')
	if strings.HasPrefix(clause, "contains(") && strings.HasSuffix(clause, ")") {
		inner := clause[len("contains(") : len(clause)-1]
		parts := splitCSV(inner)
		if len(parts) != 2 {
			return false, fmt.Errorf("contains requires 2 args")
		}
		key := trimQuotes(parts[0])
		sub := trimQuotes(parts[1])
		val := attrs[key]
		return strings.Contains(val, sub), nil
	}
	// regex_match(key,'^pat$') with safety caps
	if strings.HasPrefix(clause, "regex_match(") && strings.HasSuffix(clause, ")") {
		inner := clause[len("regex_match(") : len(clause)-1]
		parts := splitCSV(inner)
		if len(parts) != 2 {
			return false, fmt.Errorf("regex_match requires 2 args")
		}
		key := trimQuotes(parts[0])
		pat := trimQuotes(parts[1])
		if len(pat) > 256 {
			return false, fmt.Errorf("regex pattern too long")
		}
		rgx, ok := regexCache.compiled[pat]
		if !ok {
			compiled, err := regexp.Compile(pat)
			if err != nil {
				return false, fmt.Errorf("regex compile error: %v", err)
			}
			regexCache.compiled[pat] = compiled
			rgx = compiled
		}
		return rgx.MatchString(attrs[key]), nil
	}
	for _, op := range []string{">=", "<=", ">", "<"} {
		if strings.Contains(clause, op) {
			segs := strings.SplitN(clause, op, 2)
			key := strings.TrimSpace(segs[0])
			rhs := strings.TrimSpace(segs[1])
			valStr := attrs[key]
			if valStr == "" {
				return false, nil
			}
			lhs, err1 := strconv.ParseFloat(valStr, 64)
			rhsVal, err2 := strconv.ParseFloat(trimQuotes(rhs), 64)
			if err1 != nil || err2 != nil {
				return false, fmt.Errorf("numeric parse error in clause: %s", clause)
			}
			switch op {
			case ">":
				return lhs > rhsVal, nil
			case "<":
				return lhs < rhsVal, nil
			case ">=":
				return lhs >= rhsVal, nil
			case "<=":
				return lhs <= rhsVal, nil
			}
		}
	}
	return false, fmt.Errorf("unsupported clause: %s", clause)
}

// Utilities
func splitCSV(s string) []string {
	raw := strings.Split(s, ",")
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		if tr := strings.TrimSpace(r); tr != "" {
			out = append(out, tr)
		}
	}
	return out
}
func trimQuotes(s string) string { return strings.Trim(s, "\"'") }

func splitTopWithParens(s, delim string) []string {
	var parts []string
	depth := 0
	last := 0
	for i := 0; i < len(s); {
		switch s[i] {
		case '(':
			depth++
			i++
		case ')':
			if depth > 0 {
				depth--
			}
			i++
		default:
			if depth == 0 && strings.HasPrefix(s[i:], delim) {
				parts = append(parts, s[last:i])
				i += len(delim)
				last = i
				continue
			}
			i++
		}
	}
	parts = append(parts, s[last:])
	return parts
}

func matchingParens(s string) bool {
	if len(s) < 2 || s[0] != '(' || s[len(s)-1] != ')' {
		return false
	}
	depth := 0
	for i, ch := range s {
		if ch == '(' {
			depth++
		}
		if ch == ')' {
			depth--
			if depth == 0 && i != len(s)-1 {
				return false
			}
		}
		if depth < 0 {
			return false
		}
	}
	return depth == 0
}
