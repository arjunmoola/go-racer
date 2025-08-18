package racer

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/timer"
	"github.com/charmbracelet/bubbles/table"
	"strings"
	"os"
	"fmt"
	"slices"
	//"golang.org/x/sync/errgroup"
	"database/sql"
	"strconv"
)

type RacerState int

const (
	MAIN_MENU RacerState = iota
	SETTINGS
	GAME
	RESULTS
	STATISTICS
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
	config *Config
	stats *GameStats

	allStats table.Model
	allStatsErr error

	fileSaver chan any
	close chan struct{}
	errCh chan error

	db *sql.DB
}

type saveGameStatsRequest struct {
	stats *GameStats
}

type saveGameTestRequest struct {
	test *RacerTest
}

type saveGameStatsAndTestRequest struct {
	stats *GameStats
	test *RacerTest
}

func NewRacerModel() (*RacerModel, error) {
	model := &RacerModel{
		stateUpdateFunc: make(map[RacerState]teaUpdateFunc),
		stateViewFunc: make(map[RacerState]teaViewFunc),
		fileSaver: make(chan any),
		close: make(chan struct{}, 1),
		errCh: make(chan error, 1),
	}

	go model.listen()

	options := []string{ "start", "settings", "stats", "quit" }
	menu := &List{}
	menu.SetItems(options)

	config, err := ReadOrCreateConfig()

	if err != nil {
		return nil, err
	}

	model.config = config

	path := config.data
	path = os.ExpandEnv(path)

	_, err = os.Lstat(path)

	if err != nil {
		return nil, fmt.Errorf("invalid data path %s", path)
	}

	wordDb, err := LoadWordDb(path)

	if err != nil {
		return nil, err
	}

	stats, err := ReadGameStats()

	if err != nil {
		return nil, err
	}

	model.stats = stats

	db, err := SetupDB(defaultDbPath)

	if err != nil {
		return nil, err
	}

	_, err = GetGameStats(db)

	if err != nil {
		return nil, err
	}

	model.db = db

	game := NewGame()
	game.debug = false
	game.racer = model
	game.SetWordDb(wordDb)
	game.SetDefaultWordList(config.Words)
	game.numWordsPerLine = 20
	game.testSize = 500

	model.menu = menu
	model.game = game

	optionNames := []string{ "words", "time" }

	wordBank := make([]string, 0, len(wordDb.wordLists))

	for name := range wordDb.wordLists {
		wordBank = append(wordBank, name)
	}

	slices.Sort(wordBank)

	times := []string{"15", "25",  "30", "60", "120" }

	settingOptions := [][]string{ wordBank, times }

	settings := NewGameSettings(optionNames, settingOptions)
	settings.FromConfig(config)
	model.settings = settings
	settings.model = model

	model.allStats = table.New()

	tableCols := []table.Column{
		{ Title: "Id", Width: 10 },
		{ Title: "Name", Width: 10 },
		{ Title: "Test Duration", Width: 10 },
		{ Title: "Words", Width: 10 },
		{ Title: "Input", Width: 10 },
	}

	model.allStats.SetColumns(tableCols)

	model.registerStateUpdateFunc(MAIN_MENU, model.updateMainMenu)
	model.registerStateViewFunc(MAIN_MENU, model.viewMainMenu)
	model.registerStateUpdateFunc(GAME, model.updateGame)
	model.registerStateViewFunc(GAME, model.game.View)

	model.registerStateUpdateFunc(SETTINGS, model.updateGameSettings)
	model.registerStateViewFunc(SETTINGS, model.settings.render)

	model.registerStateUpdateFunc(STATISTICS, model.updateStats)
	model.registerStateViewFunc(STATISTICS, model.viewStats)

	model.SetState(MAIN_MENU)

	return model, nil
}

type RacerModelShutdownMsg struct{}

func (r *RacerModel) Shutdown() tea.Cmd {
	return func() tea.Msg {
		close(r.close)
		return RacerModelShutdownMsg{}
	}
}

func (r *RacerModel) listen() {
	for {
		select {
		case <-r.close:
			r.fileSaver = nil
			return
		case req := <-r.fileSaver:
			switch rq :=req.(type) {
			case saveGameStatsAndTestRequest:
				r.saveGameStatsAndTest(rq)
			case saveGameStatsRequest:
				r.saveGameStats(rq)
			}
		}
	}
}

func (r *RacerModel) saveGameStatsAndTest(rq saveGameStatsAndTestRequest) {
	//var g errgroup.Group

	//g.Go(func() error {
	//	return stats.Save()
	//})

	//g.Go(func() error {
	//	return test.Save()
	//})

	//g.Go(func() error {
	//	return InsertRacerTest(r.db, rq.test)
	//})

	//g.Go(func() error {
		//return UpdateGameStats(r.db, rq.stats)
	//})

	tx, err := r.db.Begin()

	if err != nil {
		tx.Rollback()
		r.errCh <- err
	}

	if err := UpdateGameStatsTx(tx, rq.stats); err != nil {
		tx.Rollback()
		r.errCh <- err
	}

	if err := InsertRacerTestTx(tx, rq.test); err != nil {
		tx.Rollback()
		r.errCh <- err
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		r.errCh <- err
	}
}

type saveFileErr error


func (r *RacerModel) checkErrorCmd() tea.Cmd {
	return func() tea.Msg {
		if err := <-r.errCh; err != nil {
			return saveFileErr(err)
		}

		return nil
	}
}

func (r *RacerModel) saveGameStats(rq saveGameStatsRequest) {
	//if err := rq.stats.Save(); err != nil {
	//	r.errCh <- err
	//}

	if err := UpdateGameStats(r.db, rq.stats); err != nil {
		r.errCh <- err
	}
}

func (r *RacerModel) sendSaveRequest(req any) tea.Cmd {
	return func() tea.Msg {
		r.fileSaver <- req
		return nil
	}
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
	var pcmd tea.Cmd

	switch msg :=  msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return r, r.Shutdown()
		}
	case RacerModelShutdownMsg:
		return r, tea.Quit

	case saveFileErr:
		pcmd = tea.Printf("%v\n", msg)
	}

	_, cmd := r.currentUpdateFunc(msg)

	return r, tea.Batch(cmd, r.checkErrorCmd(), pcmd)
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
		case "esc":
			return r, r.Shutdown()
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
			case "stats":
				r.SetState(STATISTICS)
				r.allStats.Focus()
				return r, r.getAllTests()
			case "quit":
				return r, r.Shutdown()
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
			g.Reset()
			r.SetState(MAIN_MENU)
			return r, nil
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

	if g.finished {
		r.stats.TotalCompleted++
		r.stats.LastTestId++

		test := &RacerTest{
			Id: r.stats.LastTestId,
			Target: g.target,
			Input: string(g.inputs),
			Test: g.test,
			Time: g.time,
		}

		stats := r.stats.Copy()

		req := saveGameStatsAndTestRequest{
			stats: stats,
			test: test,
		}

		return r, tea.Batch(cmd, timerCmd, r.sendSaveRequest(req))
	}

	return r, tea.Batch(cmd, timerCmd)
}

func (r *RacerModel) updateGameNotStarted(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	g := r.game
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			g.Reset()
			r.SetState(MAIN_MENU)
			return r, nil
		case "enter":
			g.started = true
			r.stats.Total++
			r.stats.TotalAttempted++
			g.createTest()
			cmd = g.startGame()
		}
	}

	var timerCmd tea.Cmd

	if g.started && !g.finished {
		g.timer, timerCmd = g.timer.Update(msg)
	}
	stats := r.stats.Copy()
	req := saveGameStatsRequest{ stats }
	return r, tea.Batch(cmd, timerCmd, r.sendSaveRequest(req))
}

func (r *RacerModel) updateGameFinished(msg tea.Msg) (tea.Model, tea.Cmd) {
	g := r.game
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			g.Reset()
			r.SetState(MAIN_MENU)
			return r, nil
		case "r":
			g.Reset()
			r.SetState(GAME)
			return r, nil
		case "enter":
			g.Reset()
			r.SetState(GAME)
			g.started = true
			r.stats.Total++
			r.stats.TotalAttempted++
			g.createTest()
			cmd = g.startGame()
		}
	}

	var timerCmd tea.Cmd
	var saveCmd tea.Cmd

	if g.started && !g.finished {
		g.timer, timerCmd = g.timer.Update(msg)
		stats := r.stats.Copy()
		req := saveGameStatsRequest{ stats }
		saveCmd = r.sendSaveRequest(req)
	}

	return r, tea.Batch(cmd, timerCmd, saveCmd)
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
		case "s":
			if settings.showSave {
				return r, settings.SaveSettings
			}
		case "esc":
			settings.saveSuccess = false
			settings.err = nil
			r.SetState(MAIN_MENU)
		}
	case gameSettingsSuccess:
		settings.saveSuccess = true
		settings.err = nil
		settings.showSave = false
		return r, ClearGameSettingsMessage()
	case gameSettingsErr:
		settings.saveSuccess = false
		settings.err = msg
		settings.showSave = false
		return r, ClearGameSettingsMessage()
	case clearGameSettingsMsg:
		settings.resetSaveState()
	}

	return r, nil
}

type getAllTestsErr error
type getAllTestsSuccess struct {
	tests []*RacerTest
}

func (r *RacerModel) getAllTests() tea.Cmd {
	return func() tea.Msg {
		tests, err := GetAllTests(r.db)

		if err != nil {
			return getAllTestsErr(err)
		}

		return getAllTestsSuccess{
			tests: tests,
		}
	}
}

func convertTestsToRows(tests []*RacerTest) []table.Row {
	rows := make([]table.Row, 0, len(tests))

	for _, test := range tests {
		id := strconv.Itoa(test.Id)
		time := strconv.Itoa(test.Time)
		row := append(make([]string, 0, 5), id, test.Test, time, test.Target, test.Input)

		rows = append(rows, row)
	}

	return rows

}

func (r *RacerModel) updateStats(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j":
			r.allStats.MoveDown(1)
		case "k":
			r.allStats.MoveUp(1)
		case "esc":
			r.allStats.Blur()
			r.SetState(MAIN_MENU)
			return r, nil
		}
	case getAllTestsErr:
		r.allStatsErr = msg
	case getAllTestsSuccess:
		r.allStatsErr = nil
		tests := msg.tests
		rows := convertTestsToRows(tests)
		r.allStats.SetRows(rows)
		r.allStats.Focus()
	}

	_, cmd := r.allStats.Update(msg)

	return r, cmd
}

func (r *RacerModel) viewStats() string {
	builder := &strings.Builder{}
	builder.WriteString(r.allStats.View())
	builder.WriteRune('\n')
	builder.WriteString("press esc to go back to main menu\n")
	return builder.String()
}
