package racer

import (
	"github.com/charmbracelet/lipgloss"
	"strings"
)

var (
	settingsOptionStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).Align(lipgloss.Center)
	currentSettingsOptionStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).Align(lipgloss.Center).BorderForeground(lipgloss.Color("200"))
	settingsOptionItemStyle = lipgloss.NewStyle().Padding(1)
	selectedSettingsOptionItemStyle = settingsOptionItemStyle.Foreground(lipgloss.Color("200"))
	currentSettingsOptionItemStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder())
	//selectedSettingsOptionItemStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).Foreground(lipgloss.Color("200"))
)

type settingsOption struct {
	name string
	l *List
	focus bool
}

func (o *settingsOption) render() string {
	items := make([]string, 0, len(o.l.items))

	for i, item := range o.l.items {
		//var s string
		//switch i {
		//case o.l.cursor:
		//	s = currentSettingsOptionItemStyle.Render(item)
		//case o.l.selectedIdx:
		//	s = selectedSettingsOptionItemStyle.Render(item)
		//default:
		//	s = settingsOptionItemStyle.Render(item)
		//}

		var f func(...string) string
		
		f = settingsOptionItemStyle.Render

		if !o.focus {
			if o.l.selectedIdx < 0 {
				f = settingsOptionItemStyle.Render
			} else if i == o.l.selectedIdx {
				f = selectedSettingsOptionItemStyle.Render
			}
		} else {
			if i == o.l.cursor {
				f = currentSettingsOptionItemStyle.Render
			}

			if i == o.l.selectedIdx {
				f = selectedSettingsOptionItemStyle.Render
			}

			if i == o.l.cursor && i == o.l.selectedIdx {
				f = selectedSettingsOptionItemStyle.BorderStyle(lipgloss.NormalBorder()).UnsetPadding().Render
			}
		}

		items = append(items, f(item))
	}

	s := o.name + "\n" +  lipgloss.JoinHorizontal(lipgloss.Center, items...)

	if o.focus {
		return currentSettingsOptionStyle.Render(s)
	} else {
		return settingsOptionStyle.Render(s)
	}
}

type GameSettings struct {
	options []*settingsOption
	idx int
	selectedIdx int
	model *RacerModel
	inFocus bool

	selectedOptions map[string]string
}

func (s *GameSettings) Next() {
	if s.idx + 1 < len(s.options) {
		s.options[s.idx].focus = false
		s.idx++
		s.options[s.idx].focus = true
	}
}

func (s *GameSettings) Prev() {
	if s.idx - 1 > -1 {
		s.options[s.idx].focus = false
		s.idx--
		s.options[s.idx].focus = true
	}
}

func (s *GameSettings) EnterOption() {
	s.selectedIdx = s.idx
	s.inFocus = true
	s.options[s.selectedIdx].focus = true
}

func (s *GameSettings) ExitOption() {
	s.options[s.selectedIdx].focus = false
	s.inFocus = false
	s.selectedIdx = -1
}

func (s *GameSettings) NextSettingsOption() {
	s.options[s.selectedIdx].l.Next()
}

func (s *GameSettings) PrevSettingsOption() {
	s.options[s.selectedIdx].l.Prev()
}

func (s *GameSettings) SelectSettingsOption() {
	opt := s.options[s.selectedIdx]
	opt.l.SetSelection()
	s.selectedOptions[opt.name] = opt.l.selectedValue
}

func NewGameSettings(optionNames []string, options [][]string) *GameSettings {
	settingsOptions := make([]*settingsOption, 0, len(optionNames))

	for i, name := range optionNames {
		l := NewList()
		l.SetItems(options[i])

		option := &settingsOption{
			name: name,
			l: l,
			focus: false,
		}

		settingsOptions = append(settingsOptions, option)
	}

	gameSettings := &GameSettings{
		options: settingsOptions,
		idx: 0,
		selectedIdx: 0,
		selectedOptions: make(map[string]string),
	}

	gameSettings.EnterOption()

	return gameSettings
}

func (s *GameSettings) render() string {
	builder := &strings.Builder{}

	builder.WriteString("settings")
	builder.WriteRune('\n')

	for _, opt := range s.options {
		builder.WriteString(opt.render())
		builder.WriteByte('\n')
	}

	builder.WriteString("press esc to exit\n")
	builder.WriteString("press b to unfocus/go back to main menu\n")

	return builder.String()
}
