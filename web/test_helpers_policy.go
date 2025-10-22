package web

import "bytes"

// bytesReader converts a string into a *bytes.Reader for request bodies in tests.
func bytesReader(s string) *bytes.Reader { return bytes.NewReader([]byte(s)) }
