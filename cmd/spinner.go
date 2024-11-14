package cmd

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type spinnerModel struct {
	spinner  spinner.Model
	quitting bool
	message  string
	done     chan bool
}

func newSpinnerModel(message string) *spinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Moon
	return &spinnerModel{
		spinner: s,
		message: message,
		done:    make(chan bool),
	}
}

func (m *spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	default:
		return m, nil
	}
}

func (m *spinnerModel) View() string {
	if m.quitting {
		return ""
	}
	return fmt.Sprintf("\n\n   %s %s\n\n", m.spinner.View(), m.message)
}

func StartSpinner(message string) func() {
	m := newSpinnerModel(message)
	p := tea.NewProgram(m)

	// Create a cancellable context
	_, cancel := context.WithCancel(context.Background())

	// Run the spinner in a separate goroutine
	go func() {
		p.Start()
	}()

	// Return a function to stop the spinner
	return func() {
		m.quitting = true
		cancel() // Cancel the context to stop the spinner
		p.Quit() // Quit the spinner
	}
}
