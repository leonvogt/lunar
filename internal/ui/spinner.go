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
		// Clear the spinner lines when quitting
		return "\r\033[2K\033[1A\033[2K\033[1A\033[2K"
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
		m.quitting = true
		p.Quit() // Quit the spinner
		<-done   // Wait for the spinner to actually finish
		// Additional cleanup - print a carriage return to ensure clean line
		fmt.Print("\r")
	}
}
