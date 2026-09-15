package secrets

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"sync"
)

// PolicyCompiler owns a restart-scoped native selection. Its immutable native
// catalog and baseline are compiled lazily and shared by all tenant policies.
// Use NewPolicyCompiler to construct one; the zero value is not usable.
type PolicyCompiler struct {
	catalog  func() (*compiledCatalog, error)
	baseline func() (*CompiledPolicy, error)
}

var defaultPolicyCompiler = sync.OnceValues(func() (*PolicyCompiler, error) {
	return newPolicyCompiler(nil)
})

// NewPolicyCompiler validates and snapshots the operator's native selection
// without compiling expressions. A nil selection enables every supported rule;
// a non-nil empty selection enables none. Custom rules remain tenant-owned.
func NewPolicyCompiler(enabledRules *[]string) (*PolicyCompiler, error) {
	if enabledRules == nil {
		return defaultPolicyCompiler()
	}
	return newPolicyCompiler(enabledRules)
}

func newPolicyCompiler(enabledRules *[]string) (*PolicyCompiler, error) {
	selected, err := validateNativeSelection(enabledRules)
	if err != nil {
		return nil, err
	}
	all := enabledRules == nil
	count := len(nativeRuleSpecs)
	if !all {
		count = len(*enabledRules)
	}
	compiler := &PolicyCompiler{}
	compiler.catalog = sync.OnceValues(func() (*compiledCatalog, error) {
		if all {
			return compileCatalog(nativeRuleSpecs)
		}
		// Snapshotting metadata indices above avoids retaining caller slices or
		// copying native specifications until the catalog is actually needed.
		specs := make([]catalogRuleSpec, 0, count)
		for i, spec := range nativeRuleSpecs {
			if selected.has(uint16(i)) {
				specs = append(specs, spec)
			}
		}
		return compileCatalog(specs)
	})
	compiler.baseline = sync.OnceValues(func() (*CompiledPolicy, error) {
		return compiler.compilePolicy(Policy{})
	})
	return compiler, nil
}

// CompilePolicy applies tenant exclusions and custom rules within this
// compiler's native selection. It never changes the shared native catalog.
func (c *PolicyCompiler) CompilePolicy(policy Policy) (*CompiledPolicy, error) {
	if isBaselinePolicy(policy) {
		return c.baseline()
	}
	return c.compilePolicy(policy)
}

func validateNativeSelection(enabledRules *[]string) (ruleSet, error) {
	var selected ruleSet
	if len(nativeRuleSpecs) > catalogRuleWords*64 {
		return selected, errors.New("secret catalog exceeds the native rule limit")
	}
	for i, spec := range nativeRuleSpecs {
		if i > 0 && nativeRuleSpecs[i-1].ID == spec.ID {
			return selected, errors.New("secret catalog contains duplicate IDs")
		}
	}
	if enabledRules == nil {
		return selected, nil
	}
	if len(*enabledRules) > len(nativeRuleSpecs) {
		return selected, errors.New("secrets enabled_rules has too many rules")
	}
	for i, id := range *enabledRules {
		index, supported := supportedNativeRuleIndex(id)
		if len(id) > maxPolicyStringLength || !supported {
			return ruleSet{}, fmt.Errorf("enabled secrets rule %d is not supported", i+1)
		}
		if selected.has(index) {
			return ruleSet{}, fmt.Errorf("enabled secrets rule %d is duplicated", i+1)
		}
		selected.add(index)
	}
	return selected, nil
}

// Native metadata is sorted once at package initialization. Lookup requires no
// compiled catalog, including when a globally inactive ID is being validated.
func supportedNativeRuleIndex(id string) (uint16, bool) {
	index, supported := slices.BinarySearchFunc(nativeRuleSpecs, id, func(spec catalogRuleSpec, id string) int {
		return cmp.Compare(spec.ID, id)
	})
	return uint16(index), supported
}
