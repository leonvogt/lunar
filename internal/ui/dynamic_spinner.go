package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type dynamicSpinnerModel struct {
	spinner   spinner.Model
	quitting  bool
	message   string
	info      string
	startTime time.Time
}

type infoUpdateMsg string
type elapsedTickMsg time.Time

func newDynamicSpinnerModel(message string) *dynamicSpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Moon
	return &dynamicSpinnerModel{
		spinner:   s,
		message:   message,
		startTime: time.Now(),
	}
}

func elapsedTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return elapsedTickMsg(t)
	})
}

func (m *dynamicSpinnerModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, elapsedTickCmd())
}

func (m *dynamicSpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case infoUpdateMsg:
		m.info = string(msg)
		return m, nil

	case elapsedTickMsg:
		return m, elapsedTickCmd()

	default:
		return m, nil
	}
}

func (m *dynamicSpinnerModel) View() string {
	if m.quitting {
		return "\r\033[2K\033[1A\033[2K\033[1A\033[2K"
	}

	elapsed := time.Since(m.startTime).Round(time.Second)
	elapsedStr := formatDuration(elapsed)

	if m.info != "" {
		return fmt.Sprintf("\n\n   %s %s (%s) %s\n\n", m.spinner.View(), m.message, m.info, elapsedStr)
	}
	return fmt.Sprintf("\n\n   %s %s %s\n\n", m.spinner.View(), m.message, elapsedStr)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds elapsed", int(d.Seconds()))
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds elapsed", minutes, seconds)
}

// Returns an update function to set the info text, and a stop function that returns the elapsed duration.
func StartDynamicSpinner(message string) (setInfo func(info string), stop func() time.Duration) {
	m := newDynamicSpinnerModel(message)
	p := tea.NewProgram(m)

	done := make(chan bool)
	go func() {
		defer func() {
			done <- true
		}()
		p.Run()
	}()

	setInfo = func(info string) {
		p.Send(infoUpdateMsg(info))
	}

	stop = func() time.Duration {
		elapsed := time.Since(m.startTime)
		m.quitting = true
		p.Quit()
		<-done
		fmt.Print("\r")
		return elapsed
	}

	return setInfo, stop
}
