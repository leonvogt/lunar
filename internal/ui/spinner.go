package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type spinnerModel struct {
	spinner  spinner.Model
	quitting bool
	message  string
}

// quitMsg signals a spinner to tear down.
type quitMsg struct{}

func newSpinnerModel(message string) *spinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Moon
	return &spinnerModel{
		spinner: s,
		message: message,
	}
}

func (m *spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, nil

	case quitMsg:
		m.quitting = true
		return m, tea.Quit

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
		// Render nothing; bubbletea erases the spinner's own lines on teardown.
		return ""
	}
	return fmt.Sprintf("\n\n   %s %s\n\n", m.spinner.View(), m.message)
}

func StartSpinner(message string) func() {
	m := newSpinnerModel(message)
	p := tea.NewProgram(m)

	// Channel to wait for the program to finish
	done := make(chan bool)

	// Run the spinner in a separate goroutine
	go func() {
		defer func() {
			done <- true
		}()
		p.Run()
	}()

	// Return a function to stop the spinner
	return func() {
		p.Send(quitMsg{}) // Quit the spinner (handled in Update)
		<-done            // Wait for the spinner to actually finish
	}
}
