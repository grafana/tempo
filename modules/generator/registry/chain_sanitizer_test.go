package registry

import (
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
)

func TestChainSanitizer_Empty(t *testing.T) {
	s := NewChainSanitizer()
	require.IsType(t, &ChainSanitizer{}, s)
	require.Empty(t, s.sanitizers)

	lbls := labels.FromStrings("foo", "bar")
	require.Equal(t, lbls, s.Sanitize(lbls))
}

func TestChainSanitizer_NilFiltering(t *testing.T) {
	s := NewChainSanitizer(nil, nil, nil)
	require.Empty(t, s.sanitizers)
}

func TestChainSanitizer_Single(t *testing.T) {
	inner := sanitizerFunc(func(lbls labels.Labels) labels.Labels {
		return labels.NewBuilder(lbls).Set("added", "yes").Labels()
	})
	s := NewChainSanitizer(nil, inner, nil)
	require.Len(t, s.sanitizers, 1)

	result := s.Sanitize(labels.FromStrings("foo", "bar"))
	require.Equal(t, "yes", result.Get("added"))
}

func TestChainSanitizer_MultiOrderedExecution(t *testing.T) {
	var order []string

	s1 := sanitizerFunc(func(lbls labels.Labels) labels.Labels {
		order = append(order, "first")
		return labels.NewBuilder(lbls).Set("step", "1").Labels()
	})
	s2 := sanitizerFunc(func(lbls labels.Labels) labels.Labels {
		order = append(order, "second")
		require.Equal(t, "1", lbls.Get("step"))
		return labels.NewBuilder(lbls).Set("step", "2").Labels()
	})

	s := NewChainSanitizer(s1, s2)
	require.Len(t, s.sanitizers, 2)

	result := s.Sanitize(labels.FromStrings("foo", "bar"))
	require.Equal(t, "2", result.Get("step"))
	require.Equal(t, []string{"first", "second"}, order)
}

func TestChainSanitizer_NilsAmongMultiple(t *testing.T) {
	called := false
	s1 := sanitizerFunc(func(lbls labels.Labels) labels.Labels {
		called = true
		return lbls
	})

	s := NewChainSanitizer(nil, s1, nil)
	require.Len(t, s.sanitizers, 1)

	s.Sanitize(labels.FromStrings("a", "b"))
	require.True(t, called)
}
