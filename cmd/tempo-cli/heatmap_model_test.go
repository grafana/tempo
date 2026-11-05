package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/tempodb/backend"
)

func TestTextFieldHandleKey(t *testing.T) {
	f := newTextField("ab")

	edited, handled := f.handleKey(tea.KeyPressMsg{Text: "c"})
	require.True(t, edited)
	require.True(t, handled)
	require.Equal(t, "abc", f.String())

	edited, handled = f.handleKey(tea.KeyPressMsg{Code: tea.KeyLeft})
	require.False(t, edited)
	require.True(t, handled)

	edited, _ = f.handleKey(tea.KeyPressMsg{Code: tea.KeyBackspace})
	require.True(t, edited)
	require.Equal(t, "ac", f.String())

	// keys with no text and no editing meaning are not handled
	_, handled = f.handleKey(tea.KeyPressMsg{Code: tea.KeyF1})
	require.False(t, handled)
}

func TestQueryModeCycle(t *testing.T) {
	m := &heatmapModel{query: newTextField(""), tag: newTextField(defaultTag)}

	m.focus = focusTag
	for _, want := range []queryMode{modeTagNames, modeTagValues, modeSpans} {
		m.mode = (m.mode + 1) % numQueryModes
		require.Equal(t, want, m.mode)
	}
	require.False(t, m.mode.usesTag())
}

func TestTriggerTagFetchValidation(t *testing.T) {
	tests := []struct {
		name string
		mode queryMode
		tag  string
		want string
	}{
		{"unknown scope", modeTagNames, "bogus", "unknown scope"},
		{"values need a tag", modeTagValues, "", "tag is required"},
		{"bad identifier", modeTagValues, "resource.", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &heatmapModel{mode: tt.mode, tag: newTextField(tt.tag), scope: newTextField(tt.tag)}
			require.Nil(t, m.triggerTagFetch(""))
			require.NotEmpty(t, m.errText)
			require.Contains(t, m.errText, tt.want)
		})
	}
}

func TestTagNamesDefaultScopeIsValid(t *testing.T) {
	m := &heatmapModel{
		mode:  modeTagNames,
		query: newTextField(""),
		tag:   newTextField(defaultTag),
		scope: newTextField(""),
	}
	for _, q := range []string{"", "{ }"} {
		m.errText = ""
		m.triggerTagFetch(q)
		require.Empty(t, m.errText, "query %q", q)
	}
}

func TestModeModal(t *testing.T) {
	m := &heatmapModel{meta: &backend.BlockMeta{TotalObjects: 10, Size_: 1000}, width: 100, height: 30, query: newTextField(""), tag: newTextField(defaultTag), scope: newTextField("")}

	m.focus = focusMode
	m.handleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.True(t, m.modalOpen)

	m.handleKey(tea.KeyPressMsg{Code: tea.KeyDown})
	require.Equal(t, modeTagNames, m.modalSel)
	require.Equal(t, modeSpans, m.mode, "mode only changes on enter")

	out := m.overlayModal(strings.Repeat(strings.Repeat(".", 100)+"\n", 29))
	require.Contains(t, out, "Query mode")
	require.Contains(t, out, "tag values")

	m.handleKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	require.False(t, m.modalOpen)
	require.Equal(t, modeSpans, m.mode)

	m.handleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.handleKey(tea.KeyPressMsg{Code: tea.KeyDown})
	m.handleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.False(t, m.modalOpen)
	require.Equal(t, modeTagNames, m.mode)
}

func TestCycleFocus(t *testing.T) {
	m := &heatmapModel{focus: focusMode}

	// tag box is skipped in spans mode
	m.cycleFocus(1)
	require.Equal(t, focusQuery, m.focus)
	m.cycleFocus(1)
	require.Equal(t, focusMode, m.focus)
	m.cycleFocus(-1)
	require.Equal(t, focusQuery, m.focus)

	m.mode = modeTagValues
	m.focus = focusMode
	for _, want := range []fieldFocus{focusTag, focusQuery, focusMode} {
		m.cycleFocus(1)
		require.Equal(t, want, m.focus)
	}
	m.cycleFocus(-1)
	require.Equal(t, focusQuery, m.focus)
}

func TestTextFieldRenderWindow(t *testing.T) {
	f := newTextField("abcdefghij")
	require.Equal(t, "abcde", f.render(false, 5))
	require.Equal(t, "ghij█", f.render(true, 5))
	f.cursor = 0
	require.Equal(t, "█abcd", f.render(true, 5))
}
