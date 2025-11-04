package compliance

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// PoACondition represents a modular condition for PoA validation.
type PoACondition struct {
	Type string // e.g. "amount", "time", "custom"
	Expr string // DSL expression, e.g. ">=1000", "between(09:00,17:00)", "custom:approved"
}

// PoAValidator validates conditions using a simple DSL.
type PoAValidator struct{}

func NewPoAValidator() *PoAValidator {
	return &PoAValidator{}
}

// Validate checks a list of PoA conditions against provided context values.
func (v *PoAValidator) Validate(conds []PoACondition, ctx map[string]interface{}) error {
	for _, cond := range conds {
		switch cond.Type {
		case "amount":
			if err := validateAmount(cond.Expr, ctx["amount"]); err != nil {
				return fmt.Errorf("amount condition failed: %w", err)
			}
		case "time":
			if err := validateTime(cond.Expr, ctx["time"]); err != nil {
				return fmt.Errorf("time condition failed: %w", err)
			}
		case "custom":
			if err := validateCustom(cond.Expr, ctx["custom"]); err != nil {
				return fmt.Errorf("custom condition failed: %w", err)
			}
		}
	}
	return nil
}

func validateAmount(expr string, val interface{}) error {
	f, ok := val.(float64)
	if !ok {
		return fmt.Errorf("amount not float64")
	}
	if strings.HasPrefix(expr, ">=") {
		min, _ := strconv.ParseFloat(strings.TrimPrefix(expr, ">="), 64)
		if f < min {
			return fmt.Errorf("%f < %f", f, min)
		}
	}
	if strings.HasPrefix(expr, "<=") {
		max, _ := strconv.ParseFloat(strings.TrimPrefix(expr, "<="), 64)
		if f > max {
			return fmt.Errorf("%f > %f", f, max)
		}
	}
	return nil
}

func validateTime(expr string, val interface{}) error {
	t, ok := val.(string)
	if !ok {
		return fmt.Errorf("time not string")
	}
	re := regexp.MustCompile(`between\((\d{2}:\d{2}),(\d{2}:\d{2})\)`)
	m := re.FindStringSubmatch(expr)
	if len(m) == 3 {
		start, end := m[1], m[2]
		if t < start || t > end {
			return fmt.Errorf("%s not in range %s-%s", t, start, end)
		}
	}
	return nil
}

func validateCustom(expr string, val interface{}) error {
	v, ok := val.(string)
	if !ok {
		return fmt.Errorf("custom not string")
	}
	if strings.HasPrefix(expr, "custom:") {
		exp := strings.TrimPrefix(expr, "custom:")
		if v != exp {
			return fmt.Errorf("custom value %s != %s", v, exp)
		}
	}
	return nil
}
