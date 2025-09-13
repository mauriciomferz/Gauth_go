package authz

import "strconv"

// NewAccessRequest creates a new access request with subject, resource, and action
func NewAccessRequest(subject Subject, resource Resource, action Action) *AccessRequest {
	return &AccessRequest{
		Subject:  subject,
		Resource: resource,
		Action:   action,
		Context:  make(map[string]string),
	}
}

// WithContext adds context information to the access request (type-safe)
func (r *AccessRequest) WithContext(context map[string]string) *AccessRequest {
	r.Context = context
	return r
}

// WithContextValue adds a single context value to the access request (as string)
func (r *AccessRequest) WithContextValue(key string, value interface{}) *AccessRequest {
	if r.Context == nil {
		r.Context = make(map[string]string)
	}
	switch v := value.(type) {
	case string:
		r.Context[key] = v
	case int:
		r.Context[key] = strconv.Itoa(v)
	case bool:
		r.Context[key] = strconv.FormatBool(v)
	case float64:
		r.Context[key] = strconv.FormatFloat(v, 'f', -1, 64)
	default:
		r.Context[key] = ""
	}
	return r
}

// WithStringValue adds a string context value to the access request
func (r *AccessRequest) WithStringValue(key string, value string) *AccessRequest {
	if r.Context == nil {
		r.Context = make(map[string]string)
	}
	r.Context[key] = value
	return r
}

// WithIntValue adds an integer context value to the access request (as string)
func (r *AccessRequest) WithIntValue(key string, value int) *AccessRequest {
	if r.Context == nil {
		r.Context = make(map[string]string)
	}
	r.Context[key] = strconv.Itoa(value)
	return r
}

// WithBoolValue adds a boolean context value to the access request (as string)
func (r *AccessRequest) WithBoolValue(key string, value bool) *AccessRequest {
	if r.Context == nil {
		r.Context = make(map[string]string)
	}
	r.Context[key] = strconv.FormatBool(value)
	return r
}

// AddAnnotation adds an annotation to the access response (as string)
func (r *AccessResponse) AddAnnotation(key string, value interface{}) {
	if r.Annotations == nil {
		r.Annotations = make(map[string]string)
	}
	switch v := value.(type) {
	case string:
		r.Annotations[key] = v
	case int:
		r.Annotations[key] = strconv.Itoa(v)
	case bool:
		r.Annotations[key] = strconv.FormatBool(v)
	case float64:
		r.Annotations[key] = strconv.FormatFloat(v, 'f', -1, 64)
	default:
		r.Annotations[key] = ""
	}
}
