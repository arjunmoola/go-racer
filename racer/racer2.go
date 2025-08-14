package racer

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/timer"
	"strings"
	"os"
)

type RacerState int

const (
	MAIN_MENU RacerState = iota
	SETTINGS
	GAME
	RESULTS
)

type teaUpdateFunc func(tea.Msg) (tea.Model, tea.Cmd)
type teaViewFunc func() string

type RacerModel struct {
	menu *List
	state RacerState
	prevState RacerState
	stateUpdateFunc map[RacerState]teaUpdateFunc
	stateViewFunc map[RacerState]teaViewFunc

	currentUpdateFunc teaUpdateFunc
	currentViewFunc teaViewFunc

	game *Game
	settings *GameSettings
}

func NewRacerModel() (*RacerModel, error) {
	model := &RacerModel{
		stateUpdateFunc: make(map[RacerState]teaUpdateFunc),
		stateViewFunc: make(map[RacerState]teaViewFunc),
	}

	options := []string{ "start", "settings", "quit" }
	menu := &List{}
	menu.SetItems(options)

	path := os.Getenv("DATA_DIR")

	if path == "" {
		return nil, ErrWordDirNotFound
	}

	wordDb, err := LoadWordDb(path)

	if err != nil {
		return nil, err
	}

	game := NewGame()
	game.racer = model
	game.SetWordDb(wordDb)
	game.SetDefaultWordList("english_1k")
	game.numWordsPerLine = 20
	game.testSize = 500

	model.menu = menu
	model.game = game

	optionNames := []string{ "words", "time" }

	wordBank := make([]string, 0, len(wordDb.wordLists))

	for name := range wordDb.wordLists {
		wordBank = append(wordBank, name)
	}

	times := []string{"15", "25",  "30", "60", "120" }

	settingOptions := [][]string{ wordBank, times }

	settings := NewGameSettings(optionNames, settingOptions)
	model.settings = settings

	model.registerStateUpdateFunc(MAIN_MENU, model.updateMainMenu)
	model.registerStateViewFunc(MAIN_MENU, model.viewMainMenu)
	model.registerStateUpdateFunc(GAME, model.updateGame)
	model.registerStateViewFunc(GAME, model.game.View)

	model.registerStateUpdateFunc(SETTINGS, model.updateGameSettings)
	model.registerStateViewFunc(SETTINGS, model.settings.render)

	model.SetState(MAIN_MENU)

	return model, nil
}

func (r *RacerModel) Run() error {
	if _, err := tea.NewProgram(r, tea.WithAltScreen()).Run(); err != nil {
		return err
	}
	return nil
}

func (r *RacerModel) State() RacerState {
	return r.state
}

func (r *RacerModel) SetState(state RacerState) {
	r.prevState = r.state
	r.state = state

	f := r.stateUpdateFunc[r.state]
	v := r.stateViewFunc[r.state]

	r.currentUpdateFunc = f
	r.currentViewFunc = v
}

func (r *RacerModel) Init() tea.Cmd {
	return nil
}

func (r *RacerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg :=  msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return r, tea.Quit
		}
	}

	return r.currentUpdateFunc(msg)

}

func (r *RacerModel) View() string {
	return r.currentViewFunc()
}

func (r *RacerModel) registerStateUpdateFunc(state RacerState, updater teaUpdateFunc) {
	r.stateUpdateFunc[state] = updater
}

func (r *RacerModel) registerStateViewFunc(state RacerState, viewer teaViewFunc) {
	r.stateViewFunc[state] = viewer
}

func (r *RacerModel) updateMainMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	menu := r.menu
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return r, tea.Quit
		case "j":
			menu.Next()
		case "k":
			menu.Prev()
		case "enter":
			menu.SetSelection()
			selectedOption := menu.SelectedValue()
			switch selectedOption {
			case "start":
				r.SetState(GAME)
			case "settings":
				r.SetState(SETTINGS)
			case "quit":
				return r, tea.Quit
			}
		}
	}
	return r, nil
}

func (r *RacerModel) viewMainMenu() string {
	menu := r.menu
	builder := &strings.Builder{}

	for idx, item := range menu.items {
		if idx == menu.cursor {
			builder.WriteString(cursorStyle.Render(item)+"\n")
		} else {
			builder.WriteString(item+"\n")
		}
	}
	return builder.String()
}

func (r *RacerModel) updateGame(msg tea.Msg) (tea.Model, tea.Cmd) {
	g := r.game
	if !g.started {
		return r.updateGameNotStarted(msg)
	}

	if g.finished {
		return r.updateGameFinished(msg)
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return g, tea.Quit
		case tea.KeyRunes:
			runes := msg.Runes
			g.charIdx = g.idx
			char := byte(runes[0])

			g.appendByte(char)
			g.incIndex()
			if len(g.target) == len(g.inputs) {
				g.finished = true
				cmd = g.stopGame()
			}
		case tea.KeyBackspace:
			if len(g.inputs) == 0 {
				break
			}
			g.trimByte()
			g.decIndex()
		case tea.KeySpace:
			g.charIdx = g.idx
			g.appendByte(' ')
			g.incIndex()
			if len(g.target) == len(g.inputs) {
				g.finished = true
				cmd = g.stopGame()
			}
		}
	case timer.TickMsg:
		if msg.Timeout {
			g.finished = true
			break
		}

		if g.started && !g.finished {
			g.ticks++
		}
	}

	var timerCmd tea.Cmd

	if g.started && !g.finished {
		g.timer, timerCmd = g.timer.Update(msg)
	}

	return r, tea.Batch(cmd, timerCmd)
}

func (r *RacerModel) updateGameNotStarted(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	g := r.game
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "b":
			g.Reset()
			r.SetState(MAIN_MENU)
			return r, nil
		case "esc":
			return r, tea.Quit
		case "enter":
			g.started = true
			g.target = g.createTest()
			cmd = g.startGame()
		}
	}

	var timerCmd tea.Cmd

	if g.started && !g.finished {
		g.timer, timerCmd = g.timer.Update(msg)
	}

	return r, tea.Batch(cmd, timerCmd)
}

func (r *RacerModel) updateGameFinished(msg tea.Msg) (tea.Model, tea.Cmd) {
	g := r.game
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "b":
			g.Reset()
			r.SetState(MAIN_MENU)
		case "esc":
			return g, tea.Quit
		case "r":
			g.Reset()
			r.SetState(GAME)
		}
	}

	return r, nil
}

func (r *RacerModel) updateGameSettings(msg tea.Msg) (tea.Model, tea.Cmd) {
	settings := r.settings
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j":
			settings.ExitOption()
			settings.Next()
			settings.EnterOption()
		case "k":
			settings.ExitOption()
			settings.Prev()
			settings.EnterOption()
		case "h":
			settings.PrevSettingsOption()
		case "l":
			settings.NextSettingsOption()
		case "enter":
			settings.SelectSettingsOption()
			
		case "b":
			r.SetState(MAIN_MENU)
		}
	}

	return r, nil
}
