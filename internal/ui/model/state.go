package model

import (
	"image/color"

	"github.com/jxdones/stoat/internal/database"
	"github.com/jxdones/stoat/internal/ui/theme"
)

// FocusedPanel indicates which panel receives key input.
type FocusedPanel int

const (
	FocusNone FocusedPanel = iota
	FocusSidebar
	FocusFilterbox
	FocusTable
	FocusQuerybox
)

// activeModal tracks the currently active modal.
type activeModal int

const (
	modalNone activeModal = iota
	modalConnectionPicker
	modalCellDetail
)

// mode tracks the current mode of the application.
type mode int

const (
	modeNormal mode = iota
	modeInsert
	modeVisual
	modeDelete
	modeCommand
)

// screenState tracks the size and focus of the main screen.
type screenState struct {
	width   int
	height  int
	focus   FocusedPanel
	compact bool
}

// tableSchema tracks the schema of the table.
type tableSchema struct {
	columns     []database.Column
	constraints []database.Constraint
	foreignKeys []database.ForeignKey
	indexes     []database.Index
}

// pageNav indicates the direction of a page navigation request.
type pageNav int

const (
	// pageNavNone means a neutral reload/jump where history should be reset
	// to the returned start cursor.
	pageNavNone pageNav = iota
	// pageNavNext means user requested forward paging; we push the current
	// page start cursor onto history so previous-page can return to it.
	pageNavNext
	// pageNavPrev means user requested backward paging; we pop one cursor from
	// history after the response arrives.
	pageNavPrev
)

// modeStyle returns the mode label and color.
func (m Model) modeStyle() (label string, color color.Color) {
	switch m.mode {
	case modeInsert:
		return "INSERT", theme.Current.ModeInsert
	case modeVisual:
		return "VISUAL", theme.Current.ModeVisual
	case modeDelete:
		return "DELETE", theme.Current.ModeDelete
	case modeCommand:
		return "COMMAND", theme.Current.ModeCommand
	default:
		return "NORMAL", theme.Current.ModeNormal
	}
}
