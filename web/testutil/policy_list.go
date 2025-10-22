package testutil

// policy_list.go enumerates valid policy bundle fixtures and provides iteration helpers.

// ValidPolicyBundleFixtures lists policy bundles considered valid.
var ValidPolicyBundleFixtures = []struct {
    Name string
    Raw  string
}{
    {Name: "PolicyBundleB1V1", Raw: PolicyBundleB1V1},
    {Name: "PolicyBundleB2V1", Raw: PolicyBundleB2V1},
}

// IterateValidPolicyBundles invokes fn for each valid policy bundle fixture; stop early if fn returns false.
func IterateValidPolicyBundles(fn func(name, raw string) bool) {
    for _, f := range ValidPolicyBundleFixtures {
        if !fn(f.Name, f.Raw) {
            return
        }
    }
}
