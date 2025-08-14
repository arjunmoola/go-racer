package racer

import (
	"os"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	"errors"
)

var (
	ErrModelNotFound = errors.New("model not found")
)

type Racer struct {
	models map[string]tea.Model
	current tea.Model
}

func NewRacer() (*Racer, error) {
	racer := &Racer{
		models: make(map[string]tea.Model),
	}

	options := []string{ "start", "settings", "quit" }
	menu := NewMenu(options)

	path := os.Getenv("DATA_DIR")

	if path == "" {
		return nil, ErrWordDirNotFound
	}

	wordDb, err := LoadWordDb(path)

	if err != nil {
		return nil, err
	}

	game := NewGame()
	target := strings.Repeat("hello world", 10)
	game.SetTarget(target)
	game.SetWordDb(wordDb)
	game.SetDefaultWordList("english_1k")
	game.numWordsPerLine = 20
	game.testSize = 500

	racer.registerModel("menu", menu)
	racer.registerModel("game", game)

	if err := racer.SetCurrent("menu"); err != nil {
		return nil, err
	}



	return racer, nil
}

func (r *Racer) SetCurrent(name string) error {
	m, ok := r.models[name]
	if !ok {
		return ErrModelNotFound
	}
	r.current = m
	return nil
}


func (r *Racer) Init() tea.Cmd {
	return nil
}

func (r *Racer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := r.current.Update(msg)

	switch msg.(type) {
	case startGameEvent:
		r.SetCurrent("game")
	case mainMenuEvent:
		r.SetCurrent("menu")
	case openSettingsEvent:
		r.SetCurrent("game")
	}
	return r, cmd

}

func (r *Racer) View() string {
	return r.current.View()
}

func (r *Racer) registerModel(name string, m tea.Model) {
	r.models[name] = m
}

func (r *Racer) Run() error {
	if _, err := tea.NewProgram(r, tea.WithAltScreen()).Run(); err != nil {
		return err
	}
	return nil
}
