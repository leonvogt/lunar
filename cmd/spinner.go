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
	switch msg.(type) {
	case tea.KeyMsg:
		m.quitting = true
		close(m.done)
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
		return ""
	}
	return fmt.Sprintf("\n\n   %s %s\n\n", m.spinner.View(), m.message)
}

func StartSpinner(message string) func() {
	m := newSpinnerModel(message)
	p := tea.NewProgram(m)

	go func() {
		_ = p.Start()
	}()

	// Stop function to end the spinner
	return func() {
		m.quitting = true
		p.Quit()
		<-m.done // Wait for spinner to fully stop
	}
}
