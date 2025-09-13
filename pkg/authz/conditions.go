package authz

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/util"
)

// TimeRangeCondition allows access only during specific time periods
type TimeRangeCondition struct {
	TimeRange *util.TimeRange
	TimeZone  *time.Location
}

func (c *TimeRangeCondition) Evaluate(ctx context.Context, request *AccessRequest) (bool, error) {
	now := time.Now()
	if c.TimeZone != nil {
		now = now.In(c.TimeZone)
	}
	allowed, _ := c.TimeRange.IsAllowed(now)
	return allowed, nil
}

// IPRangeCondition allows access only from specific IP ranges
type IPRangeCondition struct {
	AllowedRanges []string
}

func (c *IPRangeCondition) Evaluate(ctx context.Context, request *AccessRequest) (bool, error) {
	if ip, ok := request.Context["ip"]; ok {
		return c.isIPInRanges(ip), nil
	}
	return false, nil
}

func (c *IPRangeCondition) isIPInRanges(ip string) bool {
	// Implement IP range checking
	return false
}

// ResourceOwnerCondition allows access only to resource owners
type ResourceOwnerCondition struct {
	OwnerIDField string
}

func (c *ResourceOwnerCondition) Evaluate(ctx context.Context, request *AccessRequest) (bool, error) {
	if ownerID, ok := request.Context[c.OwnerIDField]; ok {
		return request.Subject.ID == ownerID, nil
	}
	return false, nil
}

// RoleCondition allows access based on user roles
type RoleCondition struct {
	RequiredRoles []Role
}

func (c *RoleCondition) Evaluate(ctx context.Context, request *AccessRequest) (bool, error) {
	if rolesStr, ok := request.Context["roles"]; ok {
		roles := strings.Split(rolesStr, ",")
		for _, required := range c.RequiredRoles {
			found := false
			for _, role := range roles {
				if role == string(required) {
					found = true
					break
				}
			}
			if !found {
				return false, nil
			}
		}
		return true, nil
	}
	return false, nil
}

// AttributeCondition allows access based on resource attributes
type AttributeCondition struct {
	Attribute string
	Value     interface{}
	Operator  string // eq, ne, gt, lt, contains, etc.
}

func (c *AttributeCondition) Evaluate(ctx context.Context, request *AccessRequest) (bool, error) {
	if attr, ok := request.Context[c.Attribute]; ok {
		attrStr := attr
		valueStr := ""
		switch v := c.Value.(type) {
		case string:
			valueStr = v
		case int:
			valueStr = strconv.Itoa(v)
		case bool:
			valueStr = strconv.FormatBool(v)
		case float64:
			valueStr = strconv.FormatFloat(v, 'f', -1, 64)
		default:
			valueStr = ""
		}
		switch c.Operator {
		case "eq":
			return attrStr == valueStr, nil
		case "ne":
			return attrStr != valueStr, nil
		case "contains":
			return strings.Contains(attrStr, valueStr), nil
		default:
			return false, nil
		}
	}
	return false, nil
}

// MultiCondition combines multiple conditions with AND/OR logic
type MultiCondition struct {
	Conditions []Condition
	Operator   string // "and" or "or"
}

func (c *MultiCondition) Evaluate(ctx context.Context, request *AccessRequest) (bool, error) {
	if len(c.Conditions) == 0 {
		return true, nil
	}

	switch c.Operator {
	case "and":
		for _, cond := range c.Conditions {
			allowed, err := cond.Evaluate(ctx, request)
			if err != nil {
				return false, err
			}
			if !allowed {
				return false, nil
			}
		}
		return true, nil

	case "or":
		for _, cond := range c.Conditions {
			allowed, err := cond.Evaluate(ctx, request)
			if err != nil {
				return false, err
			}
			if allowed {
				return true, nil
			}
		}
		return false, nil

	default:
		return false, nil
	}
}
