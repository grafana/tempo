package regexp

import (
	"testing"
	"unsafe"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
)

func TestRegexpMatch(t *testing.T) {
	r, err := NewRegexp([]string{"^a.*"}, true)
	require.NoError(t, err)

	require.True(t, r.Match([]byte("abc")))
	require.True(t, r.MatchString("abc"))
	require.False(t, r.MatchString("xyz"))

	r, err = NewRegexp([]string{"^a.*"}, false)
	require.NoError(t, err)

	require.False(t, r.Match([]byte("abc")))
	require.False(t, r.MatchString("abc"))
	require.True(t, r.MatchString("xyz"))
}

func TestCheatToSeeInternalsSafety(t *testing.T) {
	// This test checks that our cheatToSeeInternals struct has compatible memory layout
	// with Prometheus labels.FastRegexMatcher, in case the latter changes in the future.

	t.Run("field_has_correct_order", func(t *testing.T) {
		// This test validates that critical fields are at expected offsets to detect field reordering.
		// Field reordering would pass size/alignment checks but cause reading wrong memory locations.

		var cheat cheatToSeeInternals

		require.Equal(t, uintptr(0), unsafe.Offsetof(cheat.reString),
			"reString should be at offset 0 - struct field order may have changed")

		stringMatcherOffset := unsafe.Offsetof(cheat.stringMatcher)
		setMatchesOffset := unsafe.Offsetof(cheat.setMatches)
		prefixOffset := unsafe.Offsetof(cheat.prefix)
		suffixOffset := unsafe.Offsetof(cheat.suffix)

		require.True(t, stringMatcherOffset > setMatchesOffset,
			"stringMatcher should come after setMatches")
		require.True(t, prefixOffset > stringMatcherOffset,
			"prefix should come after stringMatcher")
		require.True(t, suffixOffset > prefixOffset,
			"suffix should come after prefix")
	})

	t.Run("struct_layout_validation", func(t *testing.T) {
		var matcher labels.FastRegexMatcher

		var cheat cheatToSeeInternals

		matcherSize := unsafe.Sizeof(matcher)
		cheatSize := unsafe.Sizeof(cheat)
		require.Equal(t, matcherSize, cheatSize, "struct size mismatch")

		matcherAlign := unsafe.Alignof(matcher)
		cheatAlign := unsafe.Alignof(cheat)
		require.Equal(t, matcherAlign, cheatAlign, "struct alignment mismatch")
	})

	t.Run("exact_string_should_have_reString_field_populated", func(t *testing.T) {
		matcher, err := labels.NewFastRegexMatcher("simple-exact-match")
		require.NoError(t, err, "should create matcher for simple-exact-match")

		cheat := (*cheatToSeeInternals)(unsafe.Pointer(matcher))

		require.Equal(t, "simple-exact-match", cheat.reString, "reString field should contain the original regex")

		require.NotNil(t, cheat.matchString, "matchString function should not be nil")

		require.True(t, cheat.matchString("simple-exact-match"), "matchString should work for exact match")
		require.False(t, cheat.matchString("different"), "matchString should reject non-matches")
	})

	t.Run("prefix_pattern_should_have_prefix_field_set", func(t *testing.T) {
		matcher, err := labels.NewFastRegexMatcher("prefix-test.*")
		require.NoError(t, err, "should create matcher for prefix-test.*")

		cheat := (*cheatToSeeInternals)(unsafe.Pointer(matcher))

		require.Equal(t, "prefix-test.*", cheat.reString, "reString should match original")
		require.Equal(t, "prefix-test", cheat.prefix, "prefix field should be set for prefix patterns")
		require.Empty(t, cheat.suffix, "suffix should be empty for prefix patterns")

		require.NotNil(t, cheat.matchString, "matchString should be available")
		require.True(t, cheat.matchString("prefix-test-something"), "should match prefix pattern")
		require.False(t, cheat.matchString("different-prefix"), "should not match different prefix")
	})

	t.Run("suffix_pattern_should_have_suffix_field_set", func(t *testing.T) {
		matcher, err := labels.NewFastRegexMatcher(".*suffix-test")
		require.NoError(t, err, "should create matcher for .*suffix-test")

		cheat := (*cheatToSeeInternals)(unsafe.Pointer(matcher))

		require.Equal(t, ".*suffix-test", cheat.reString, "reString should match original")
		require.Equal(t, "suffix-test", cheat.suffix, "suffix field should be set for suffix patterns")
		require.Empty(t, cheat.prefix, "prefix should be empty for suffix patterns")

		require.NotNil(t, cheat.matchString, "matchString should be available")
		require.True(t, cheat.matchString("something-suffix-test"), "should match suffix pattern")
		require.False(t, cheat.matchString("suffix-test-extra"), "should not match with extra suffix")
	})

	t.Run("alternation_pattern_should_populate_setMatches", func(t *testing.T) {
		matcher, err := labels.NewFastRegexMatcher("option1|option2|option3")
		require.NoError(t, err, "should create matcher for option1|option2|option3")

		cheat := (*cheatToSeeInternals)(unsafe.Pointer(matcher))

		require.Equal(t, "option1|option2|option3", cheat.reString, "reString should match original")

		expectedOptions := []string{"option1", "option2", "option3"}
		for _, expected := range expectedOptions {
			found := false
			for _, actual := range cheat.setMatches {
				if actual == expected {
					found = true
					break
				}
			}
			if !found {
				t.Logf("expected option %q not found in setMatches %v", expected, cheat.setMatches)
			}
		}

		require.NotNil(t, cheat.matchString, "matchString should be available")
		require.True(t, cheat.matchString("option1"), "should match first option")
		require.True(t, cheat.matchString("option2"), "should match second option")
		require.True(t, cheat.matchString("option3"), "should match third option")
		require.False(t, cheat.matchString("option4"), "should not match non-option")
	})

	t.Run("contains_pattern_should_populate_contains_field", func(t *testing.T) {
		matcher, err := labels.NewFastRegexMatcher(".*contains.*test.*")
		require.NoError(t, err, "should create matcher for .*contains.*test.*")

		cheat := (*cheatToSeeInternals)(unsafe.Pointer(matcher))

		require.Equal(t, ".*contains.*test.*", cheat.reString, "reString should match original")

		require.Contains(t, cheat.contains, "contains", "should include 'contains' substring")
		require.Contains(t, cheat.contains, "test", "should include 'test' substring")

		require.NotNil(t, cheat.matchString, "matchString should be available")
		require.True(t, cheat.matchString("prefix-contains-middle-test-suffix"), "should match when both substrings present")
		require.False(t, cheat.matchString("contains-only"), "should not match with only first substring")
		require.False(t, cheat.matchString("test-only"), "should not match with only second substring")
	})
}

func TestShouldMemoize(t *testing.T) {
	tcs := []struct {
		regex         string
		shouldMemoize bool
	}{
		{
			regex:         ".*",
			shouldMemoize: false,
		},
		{
			regex:         "foo.*",
			shouldMemoize: false,
		},
		{
			regex:         ".*bar",
			shouldMemoize: false,
		},
		{
			regex:         "foo|bar",
			shouldMemoize: false,
		},
		{
			regex:         ".*bar.*", // creates a containsStringMatcher so should not memoize
			shouldMemoize: false,
		},
		{
			regex:         ".*bar.*foo.*", // calls contains in order
			shouldMemoize: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.regex, func(t *testing.T) {
			m, err := labels.NewFastRegexMatcher(tc.regex)
			require.NoError(t, err)
			require.Equal(t, tc.shouldMemoize, shouldMemoize(m))
		})
	}
}
