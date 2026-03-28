package model

import "github.com/jxdones/stoat/internal/database"

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
