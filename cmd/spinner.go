package cmd

import (
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

// StartSpinner runs the spinner in a background goroutine
func StartSpinner(message string) func() chan bool {
	m := newSpinnerModel(message)
	p := tea.NewProgram(m)

	done := make(chan bool)

	go func() {
		if err := p.Start(); err != nil {
			fmt.Println("Error starting program:", err)
		}
		done <- true // Notify when the spinner is finished
		close(done)  // Close the done channel
	}()

	// Return the stop function to terminate the spinner
	return func() chan bool {
		m.quitting = true
		p.Quit()
		return done // Return the channel to wait for completion
	}
}
