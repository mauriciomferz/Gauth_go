package test

import (
	"reflect"
	"testing"
	"unsafe"
)

// importReflectSet sets the unexported receiptStore field on BetaServer.
// It uses unsafe to bypass unexported field restriction purely for test injection.
func importReflectSet(t *testing.T, srv interface{}, store interface{}) {
	t.Helper()
	v := reflect.ValueOf(srv)
	if v.Kind() != reflect.Ptr {
		t.Fatalf("srv not pointer")
	}
	ve := v.Elem()
	f := ve.FieldByName("receiptStore")
	if !f.IsValid() {
		t.Fatalf("receiptStore field not found on server struct")
	}
	if f.CanSet() {
		f.Set(reflect.ValueOf(store))
		return
	}
	// Make writable via unsafe pointer
	rf := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
	rf.Set(reflect.ValueOf(store))
}
