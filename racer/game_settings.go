package racer

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"strconv"
	"os"
	"fmt"
	"time"
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

	showSave bool
	saveSuccess bool
	err error
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
	currentSelectedValue := opt.l.SelectedValue()
	opt.l.SetSelection()

	if opt.l.SelectedValue() != currentSelectedValue {
		s.showSave = true
	}

	s.selectedOptions[opt.name] = opt.l.selectedValue

	s.updateConfig()
}

func (s *GameSettings) GetSelectedOption(key string) (string, bool) {
	value, ok := s.selectedOptions[key]
	return value, ok
}

func (s *GameSettings) updateConfig() {
	config := s.model.config

	for optName, value := range s.selectedOptions {
		switch optName {
		case "words":
			config.Words = value
		case "time":
			t, _ := strconv.ParseInt(value, 10, 64)
			config.Time = int(t)
		case "allow backspace":
			config.AllowBackspace = value == "yes"
		case "mode":
			config.GameMode = value
		case "words test size":
			t, _ := strconv.ParseInt(value, 10, 64)
			config.WordsTestSize = int(t)
		}
	}
}

type gameSettingsErr error
type gameSettingsSuccess struct{}
type clearGameSettingsMsg struct{}

func (s *GameSettings) SaveSettings() tea.Msg {
	config := s.model.config
	path := os.ExpandEnv(defaultConfigPath)

	s.updateConfig()

	file, err := os.Create(path)

	if err != nil {
		return gameSettingsErr(err)
	}

	defer file.Close()

	if err := config.write(file); err != nil {
		return gameSettingsErr(err)
	}

	return gameSettingsSuccess{}
}

func ClearGameSettingsMessage() tea.Cmd {
	return tea.Tick(1*time.Second, func(_ time.Time) tea.Msg {
		return clearGameSettingsMsg{}
	})
}

func (s *GameSettings) SetSelectedOption(name, option string) {
	for _, opt := range s.options {
		if opt.name == name {
			for i, optName := range opt.l.items {
				if optName == option {
					opt.l.selectedIdx = i
					opt.l.selectedValue = optName
					s.selectedOptions[name] = option
					return
				}
			}
		}
	}
}

func (s *GameSettings) FromConfig(config *Config) {
	t := strconv.Itoa(config.Time)
	s.SetSelectedOption("time", t)
	s.SetSelectedOption("words", config.Words)
	var allowBack string
	if config.AllowBackspace {
		allowBack = "yes"
	} else {
		allowBack = "no"
	}
	s.SetSelectedOption("allow backspace", allowBack)
	s.SetSelectedOption("mode", config.GameMode)
	s.SetSelectedOption("words test size", strconv.Itoa(config.WordsTestSize))
}

func (s *GameSettings) resetSaveState() {
	s.showSave = false
	s.saveSuccess = false
	s.err = nil
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

func (s *GameSettings) containsOption(optName string) bool {
	for _, opt := range s.options {
		if opt.name == optName {
			return true
		}
	}
	return false
}

func (s *GameSettings) registerOptions(optName string, options []string) {
	if s.containsOption(optName) {
		return
	}

	l := NewList()
	l.SetItems(options)

	opt := &settingsOption{
		name: optName,
		l: l,
		focus: false,
	}

	s.options = append(s.options, opt)

	if len(s.options) == 1 {
		s.EnterOption()
	}
}

func (s *GameSettings) render() string {
	builder := &strings.Builder{}

	builder.WriteString("settings")
	builder.WriteRune('\n')

	for _, opt := range s.options {
		builder.WriteString(opt.render())
		builder.WriteByte('\n')
	}

	builder.WriteString("press esc to go back to main menu\n")
	builder.WriteString("press ctrl+c to exit\n")
	builder.WriteRune('\n')

	if s.showSave {
		builder.WriteString("press s to save settings to disk\n")
	}

	if s.saveSuccess {
		builder.WriteString("settings saved to .go-racer/config.json\n")
	}

	if s.err != nil {
		builder.WriteString("unable to save to disk\n")
		fmt.Fprintf(builder, "got: %v\n", s.err)
	}

	return builder.String()
}
