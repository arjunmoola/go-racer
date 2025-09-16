package clock

import (
	tea "github.com/charmbracelet/bubbletea"
	"fmt"
	"time"
	"strings"
	"github.com/charmbracelet/lipgloss"
)

const defaultTickRate = time.Second

var (
	defaultClockStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("200"))
)

type clockTickMsg time.Time

func clockTick() tea.Cmd {
	return tea.Every(defaultTickRate, func(t time.Time) tea.Msg {
		return clockTickMsg(t)
	})
}

type Model struct {
	time time.Time
	ticks int
	builder *strings.Builder
	style lipgloss.Style
}

func New() Model {
	return Model{
		time: time.Now(),
		builder: &strings.Builder{},
	}
}

func (m *Model) SetStyle(style lipgloss.Style) {
	m.style = style
}

func (m Model) Init() tea.Cmd {
	return clockTick()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clockTickMsg:
		m.time = time.Time(msg)
		return m, clockTick()
	}
	return m, nil
}

func (m Model) View() string {
	m.builder.Reset()
	render := m.style.Render
	fmt.Fprintf(m.builder, "%s", render(m.time.Format(time.Kitchen)))
	return m.builder.String()
}
