// Package conditions provides conditional expression evaluation for policy rules (AAP-001 sec3.item4).
package conditions

import (
	"fmt"
	"strconv"
	"strings"
)

// Condition represents a conditional expression.
type Condition struct {
	Expression string                 `json:"expression"`
	Variables  map[string]interface{} `json:"variables"`
}

// Evaluator evaluates conditional expressions.
type Evaluator interface {
	// Evaluate evaluates a condition and returns true/false.
	Evaluate(condition *Condition) (bool, error)

	// EvaluateExpression evaluates a raw expression string with variables.
	EvaluateExpression(expression string, variables map[string]interface{}) (bool, error)
}

// SimpleEvaluator provides basic conditional expression evaluation.
type SimpleEvaluator struct {
	operators map[string]operatorFunc
}

type operatorFunc func(left, right interface{}) (bool, error)

// NewSimpleEvaluator creates a new evaluator with standard operators.
func NewSimpleEvaluator() *SimpleEvaluator {
	eval := &SimpleEvaluator{
		operators: make(map[string]operatorFunc),
	}

	// Register operators
	eval.operators["=="] = equals
	eval.operators["!="] = notEquals
	eval.operators[">"] = greaterThan
	eval.operators["<"] = lessThan
	eval.operators[">="] = greaterOrEqual
	eval.operators["<="] = lessOrEqual
	eval.operators["contains"] = contains
	eval.operators["in"] = inList

	return eval
}

// Evaluate evaluates a condition.
func (e *SimpleEvaluator) Evaluate(condition *Condition) (bool, error) {
	return e.EvaluateExpression(condition.Expression, condition.Variables)
}

// EvaluateExpression evaluates an expression with variables.
func (e *SimpleEvaluator) EvaluateExpression(expression string, variables map[string]interface{}) (bool, error) {
	// Simple expression parser: "variable operator value"
	// Examples: "age > 18", "role == admin", "country in [US,CA]"

	expression = strings.TrimSpace(expression)

	// Handle logical operators (AND, OR)
	if strings.Contains(expression, " AND ") {
		return e.evaluateAnd(expression, variables)
	}
	if strings.Contains(expression, " OR ") {
		return e.evaluateOr(expression, variables)
	}

	// Parse simple comparison
	for op := range e.operators {
		if strings.Contains(expression, " "+op+" ") {
			parts := strings.SplitN(expression, " "+op+" ", 2)
			if len(parts) != 2 {
				return false, fmt.Errorf("invalid expression format: %s", expression)
			}

			leftStr := strings.TrimSpace(parts[0])
			rightStr := strings.TrimSpace(parts[1])

			// Resolve variables
			left := resolveVariable(leftStr, variables)
			right := resolveVariable(rightStr, variables)

			// Apply operator
			handler := e.operators[op]
			return handler(left, right)
		}
	}

	return false, fmt.Errorf("no operator found in expression: %s", expression)
}

func (e *SimpleEvaluator) evaluateAnd(expression string, variables map[string]interface{}) (bool, error) {
	parts := strings.Split(expression, " AND ")
	for _, part := range parts {
		result, err := e.EvaluateExpression(strings.TrimSpace(part), variables)
		if err != nil {
			return false, err
		}
		if !result {
			return false, nil // Short-circuit
		}
	}
	return true, nil
}

func (e *SimpleEvaluator) evaluateOr(expression string, variables map[string]interface{}) (bool, error) {
	parts := strings.Split(expression, " OR ")
	for _, part := range parts {
		result, err := e.EvaluateExpression(strings.TrimSpace(part), variables)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil // Short-circuit
		}
	}
	return false, nil
}

func resolveVariable(name string, variables map[string]interface{}) interface{} {
	// Check if it's a variable reference
	if value, exists := variables[name]; exists {
		return value
	}

	// Try to parse as literal
	if num, err := strconv.ParseFloat(name, 64); err == nil {
		return num
	}

	// Return as string literal
	return strings.Trim(name, "\"'")
}

// Operator implementations

func equals(left, right interface{}) (bool, error) {
	return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right), nil
}

func notEquals(left, right interface{}) (bool, error) {
	eq, _ := equals(left, right)
	return !eq, nil
}

func greaterThan(left, right interface{}) (bool, error) {
	l, r, err := toFloat(left, right)
	if err != nil {
		return false, err
	}
	return l > r, nil
}

func lessThan(left, right interface{}) (bool, error) {
	l, r, err := toFloat(left, right)
	if err != nil {
		return false, err
	}
	return l < r, nil
}

func greaterOrEqual(left, right interface{}) (bool, error) {
	l, r, err := toFloat(left, right)
	if err != nil {
		return false, err
	}
	return l >= r, nil
}

func lessOrEqual(left, right interface{}) (bool, error) {
	l, r, err := toFloat(left, right)
	if err != nil {
		return false, err
	}
	return l <= r, nil
}

func contains(left, right interface{}) (bool, error) {
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)
	return strings.Contains(leftStr, rightStr), nil
}

func inList(left, right interface{}) (bool, error) {
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)

	// Parse list format: [item1,item2,item3]
	rightStr = strings.Trim(rightStr, "[]")
	items := strings.Split(rightStr, ",")

	for _, item := range items {
		if strings.TrimSpace(item) == leftStr {
			return true, nil
		}
	}

	return false, nil
}

func toFloat(left, right interface{}) (float64, float64, error) {
	l, err := parseFloat(left)
	if err != nil {
		return 0, 0, fmt.Errorf("left operand not numeric: %v", left)
	}

	r, err := parseFloat(right)
	if err != nil {
		return 0, 0, fmt.Errorf("right operand not numeric: %v", right)
	}

	return l, r, nil
}

func parseFloat(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert to float: %v", value)
	}
}
