package main

import (
	tea "charm.land/bubbletea/v2"

	"github.com/jxdones/stoat/internal/ui/model"
)

func main() {
	program := tea.NewProgram(model.New())
	_, _ = program.Run()
}
