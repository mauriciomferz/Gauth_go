//go:build race
// +build race

package revocation

func init() {
	raceEnabled = true
}
