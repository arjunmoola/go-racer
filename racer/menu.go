package racer

import (
	tea "github.com/charmbracelet/bubbletea"
)

type Menu struct {
	l *List
}

func NewMenu(options []string) *Menu {
	menu := NewList()
	menu.SetItems(options)
	return &Menu{
		l: menu,
	}
}

func (m *Menu) Next() {
	m.l.Next()
}

func (m *Menu) Prev() {
	m.l.Prev()
}

type startGameEvent struct{}
type openSettingsEvent struct{}
type mainMenuEvent struct{}

func (m *Menu) startGame() tea.Msg {
	return startGameEvent{}
}

func (m *Menu) openSettings() tea.Msg {
	return openSettingsEvent{}
}

func (m *Menu) Init() tea.Cmd {
	return nil
}

func (m *Menu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "enter":
			m.l.SetSelection()

			switch m.l.selectedValue {
			case "start":
				return m, m.startGame
			case "settings":
				return m, m.openSettings
			case "quit":
				return m, tea.Quit
			}
		}
	}
	_, cmd := m.l.Update(msg)
	return m, cmd
}

func (m *Menu) View() string {
	return m.l.View()
}
