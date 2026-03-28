package model

import (
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
)

// SavedQuery holds a named SQL snippet that can be expanded in the query box
// by typing @Name and triggering expansion (e.g. Ctrl+N).
type SavedQuery struct {
	Name  string
	Query string
}

// ExpandSavedQuery looks at the query box text to the left of the cursor for
// an @word token. If found and the name matches a saved query, it replaces
// @word with that query's SQL and moves the cursor to the end of the inserted text.
func (m Model) ExpandSavedQuery() (Model, bool) {
	value := m.querybox.Value()
	runes := []rune(value)

	info := m.querybox.LineInfo()
	cursorPosition := flatOffset(runes, info.RowOffset, info.CharOffset)

	name, tokenStart := findAtToken(runes, cursorPosition)
	if name == "" {
		return m, false
	}

	sql, found := lookupSavedQuery(m.savedQueries, name)
	if !found {
		return m, false
	}
	// Trim trailing newlines so the cursor stays on the last line of the snippet
	// (YAML block scalars often add one).
	sql = strings.TrimRight(sql, "\n")

	sqlRunes := []rune(sql)
	newRunes := make([]rune, len(runes)+len(sqlRunes))
	newRunes = append(newRunes, runes[:tokenStart]...)
	newRunes = append(newRunes, sqlRunes...)
	newRunes = append(newRunes, runes[cursorPosition:]...)

	newCursorPosition := tokenStart + len(sqlRunes)
	m.querybox.SetValue(string(newRunes))
	m.querybox.AdvanceCursor(newCursorPosition - cursorPosition)

	return m, true
}

// flatOffset converts a (row, offset) position into a single index into runes.
//
// The buffer is treated as lines split by '\n'. row is the 0-based line number;
// offset is the number of runes from the start of that line to the position
// The result is the rune index at that position, or len(runes)
// if the position is past the end of the text.
func flatOffset(runes []rune, row, offset int) int {
	currentRow := 0
	for i, r := range runes {
		if currentRow == row {
			return i + offset
		}
		if r == '\n' {
			currentRow++
		}
	}
	return len(runes)
}

// findAtToken finds the @word token that ends at or just before the cursor.
// It scans left from (position - 1). A word is runes [a-zA-Z0-9_].
// Returns the word and the index of the '@', or ("", -1) if no such token exists.
func findAtToken(runes []rune, position int) (string, int) {
	if position == 0 {
		return "", -1
	}

	atIndex := position - 1
	for atIndex >= 0 && isWordRune(runes[atIndex]) {
		atIndex--
	}

	if atIndex < 0 || runes[atIndex] != '@' {
		return "", -1
	}

	word := string(runes[atIndex+1 : position])
	if word == "" {
		return "", -1
	}

	return word, atIndex
}

// isWordRune reports whether r is allowed in a saved query name: letter, digit, or underscore.
func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// lookupSavedQuery finds a saved query by name (case-insensitive).
// It returns the query's SQL and true, or ("", false) if not found.
func lookupSavedQuery(queries []SavedQuery, name string) (string, bool) {
	for _, query := range queries {
		if strings.EqualFold(query.Name, name) {
			return query.Query, true
		}
	}
	return "", false
}

// handleSavedQueryKey handles ctrl+n when the querybox is focused: tries to expand a saved query.
func (m Model) handleSavedQueryKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if msg.String() != "ctrl+n" || !m.isFocused(FocusQuerybox) {
		return m, nil, false
	}
	next, expanded := m.ExpandSavedQuery()
	if !expanded {
		return m, nil, false
	}
	next.applyViewState()
	return next, nil, true
}
