package racer

import (
	"bufio"
	"time"
	//"io"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/timer"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"os"
	"fmt"
	"slices"
	//"golang.org/x/sync/errgroup"
	"database/sql"
	"go-racer/models/clock"
	//"strconv"
)

var (
	racerModelTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("200"))
	leftAlignStyle = lipgloss.NewStyle().AlignHorizontal(lipgloss.Left)
)


type RacerState int

const (
	MAIN_MENU RacerState = iota
	SETTINGS
	GAME
	GAME_INTRO
	RESULTS
	STATISTICS
	PLAYER_INFO
)

type teaUpdateFunc func(tea.Msg) (tea.Model, tea.Cmd)
type teaViewFunc func() string

type RacerModel struct {
	width int
	height int
	menu *List
	state RacerState
	prevState RacerState
	stateUpdateFunc map[RacerState]teaUpdateFunc
	stateViewFunc map[RacerState]teaViewFunc
	clock clock.Model

	currentUpdateFunc teaUpdateFunc
	currentViewFunc teaViewFunc

	game *Game
	settings *GameSettings
	config *Config
	stats *GameStats

	wordDb *WordDb
	selectedWordList *WordList

	allStats table.Model
	allStatsErr error

	fileSaver chan any
	close chan struct{}
	errCh chan error

	playerInfo *PlayerInfo
	playerFound bool
	playerInfoModel *PlayerInfoModel

	introModel *IntroModel

	db *sql.DB
	insertTestStmt *sql.Stmt
	getAllTestsStmt *sql.Stmt
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

type savePlayerInfoRequest struct {
	name string
}

func readIntroText() ([]string, error) {
	path := os.ExpandEnv("$HOME/.go-racer/wuxia/intro.txt")

	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	var chunks []string

	builder := &strings.Builder{}

	for scanner.Scan() {
		text := scanner.Text()
		if len(text) == 0 {
			chunks = append(chunks, builder.String())
			builder.Reset()
		} else {
			fmt.Fprintln(builder, text)
		}
	}

	chunks = append(chunks, builder.String())

	if len(chunks) == 0 {
		panic("incorrect number of chunks")
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return chunks, nil
}

func NewRacerModel() (*RacerModel, error) {
	model := &RacerModel{
		clock: clock.New(),
		stateUpdateFunc: make(map[RacerState]teaUpdateFunc),
		stateViewFunc: make(map[RacerState]teaViewFunc),
		fileSaver: make(chan any),
		close: make(chan struct{}, 1),
		errCh: make(chan error, 1),
	}

	go model.listen()

	options := []string{ "start", "begin", "settings", "stats", "quit" }
	menu := &List{}
	menu.SetItems(options)

	model.menu = menu

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

	chunks, err := readIntroText()

	if err != nil {
		return nil, err
	}

	model.introModel = NewIntroModel(chunks)

	wordDb, err := LoadWordDb(path)

	if err != nil {
		return nil, err
	}

	model.wordDb = wordDb
	model.selectedWordList = wordDb.wordLists[config.Words]

	stats, err := ReadGameStats()

	if err != nil {
		return nil, err
	}

	model.stats = stats

	db, err := SetupDB(defaultDbPath)

	if err != nil {
		return nil, err
	}
	model.db = db

	_, err = GetGameStats(db)

	if err != nil {
		return nil, err
	}

	playerInfo, found, err := GetPlayerInfo(db)

	if err != nil {
		return nil, err
	}

	insertTestStmt, err := prepareStatement(insertTestStmtStr, db)

	if err != nil {
		return nil, err
	}

	model.insertTestStmt = insertTestStmt

	getAllTestsQueryStmt, err := prepareStatement(getAllTestsQueryStr, db)

	if err != nil {
		return nil, err
	}
	model.getAllTestsStmt = getAllTestsQueryStmt

	model.playerInfo = playerInfo
	model.playerFound = found
	model.playerInfoModel = NewPlayerInfoModel()
	model.playerInfoModel.found = found

	if playerInfo != nil && found {
		model.playerInfoModel.value = playerInfo.name
	}

	game := NewGameFromConfig(config)
	game.racer = model
	model.game = game

	optionNames := []string{ "words", "mode", "time", "words test size", "allow backspace" }

	wordBank := make([]string, 0, len(wordDb.wordLists))

	for name := range wordDb.wordLists {
		wordBank = append(wordBank, name)
	}

	slices.Sort(wordBank)

	times := []string{"15", "25",  "30", "60", "120" }
	backspaceOptions := []string{ "yes", "no" }

	modeOptions := []string{ "time", "words" }
	wordsTestSize := []string{ "25", "50", "100" }

	settingOptions := [][]string{ wordBank, modeOptions, times, wordsTestSize, backspaceOptions }

	settings := NewGameSettings(optionNames, settingOptions)

	settings.FromConfig(config)
	model.settings = settings
	settings.model = model

	model.allStats = table.New()

	tableCols := []table.Column{
		{ Title: "Id", Width: 10 },
		{ Title: "Name", Width: 10 },
		{ Title: "Test Duration", Width: 10 },
		{ Title: "Mode", Width: 10 },
		{ Title: "Allow Backspace", Width: 10 },
		{ Title: "Test Size", Width: 10 },
		{ Title: "Accuracy", Width: 10 },
		{ Title: "Words", Width: 10 },
		{ Title: "Input", Width: 10 },
		{ Title: "Wpm", Width: 10 },
		{ Title: "Cps", Width: 10 },
		{ Title: "Rle", Width: 10 },
		{ Title: "Raw Input", Width: 10 },
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

	model.registerStateUpdateFunc(GAME_INTRO, model.updateIntroModel)
	model.registerStateViewFunc(GAME_INTRO, model.introModel.render)

	model.registerStateUpdateFunc(PLAYER_INFO, model.updatePlayerInfoModel)
	model.registerStateViewFunc(PLAYER_INFO, model.playerInfoModel.render)

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
			case savePlayerInfoRequest:
				r.insertPlayerInfo(rq)
			}
		}
	}
}

func (r *RacerModel) insertPlayerInfo(rq savePlayerInfoRequest) {
	params := &PlayerInfoInsertParams{
		name: rq.name,
	}

	err := InsertPlayerInfo(r.db, params)

	if err != nil {
		r.errCh <- err
		panic(err)
	}

	panic("here")
}

func (r *RacerModel) saveGameStatsAndTest(rq saveGameStatsAndTestRequest) {
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
type insertRacerTestErr error

func (r *RacerModel) insertRacerTestCmd(test *RacerTest) tea.Cmd {
	return func() tea.Msg {
		if err := InsertRacerTestStmt(r.insertTestStmt, test); err != nil {
			return insertRacerTestErr(err)
		}

		return nil
	}
}


func (r *RacerModel) checkErrorCmd() tea.Cmd {
	return func() tea.Msg {
		if err := <-r.errCh; err != nil {
			return saveFileErr(err)
		}

		return nil
	}
}

func (r *RacerModel) saveGameStats(rq saveGameStatsRequest) {
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
	if _, err := tea.NewProgram(r, tea.WithAltScreen(), tea.WithFPS(120)).Run(); err != nil {
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
	return tea.Batch(r.ProcessTestsCmd(), r.clock.Init())
}

type UpdateWordDb struct {
	l *WordList
}

func (r *RacerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var batch []tea.Cmd
	var cmd tea.Cmd

	var pcmd tea.Cmd
	switch msg :=  msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return r, r.Shutdown()
		}
	case RacerModelShutdownMsg:
		r.insertTestStmt.Close()
		r.getAllTestsStmt.Close()
		r.db.Close()
		return r, tea.Quit

	case insertRacerTestErr:
		pcmd = tea.Printf("%v\n", msg)
		return r, pcmd
	//case saveFileErr:
	//	pcmd = tea.Printf("%v\n", msg)
	case UpdateWordDb:
		r.wordDb.wordLists[msg.l.Name] = msg.l
		r.settings.appendSettingsOption("words", msg.l.Name)
	case tea.WindowSizeMsg:
		r.width, r.height = msg.Width, msg.Height
	}

	r.clock, cmd = r.clock.Update(msg)

	batch = append(batch, cmd)

	_, cmd = r.currentUpdateFunc(msg)

	batch = append(batch, cmd)
	batch = append(batch, r.checkErrorCmd())

	return r, tea.Batch(batch...)
}

func (r *RacerModel) View() string {
	title := racerModelTitleStyle.Render("Racer")
	clockView := lipgloss.PlaceHorizontal(r.width/2, lipgloss.Left, r.clock.View())
	titleView := lipgloss.PlaceHorizontal(r.width/2, lipgloss.Left, title)
	header := lipgloss.JoinHorizontal(lipgloss.Top, clockView, titleView)
	cView := lipgloss.Place(r.width, r.height-lipgloss.Height(header), lipgloss.Center, lipgloss.Center, leftAlignStyle.Render(r.currentViewFunc()))
	return lipgloss.JoinVertical(lipgloss.Center, header, cView)
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
			case "begin":
				r.SetState(GAME_INTRO)
				return r, doChunkTick2(string(r.introModel.lines[0]), r.introModel.idx)
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
			g.appendByte(byte(msg.Runes[0]))
			if len(g.target) == len(g.inputs) {
				g.finished = true
				cmd = g.stopGame(g.id)
			}
		case tea.KeyBackspace:
			if !g.allowBackspace {
				break
			}
			if len(g.inputs) == 0 {
				break
			}
			g.trimByte()
		case tea.KeySpace:
			g.appendByte(' ')
			if len(g.target) == len(g.inputs) {
				g.finished = true
				cmd = g.stopGame(g.id)
			}
		case tea.KeyTab:
			g.restart()
			cmd = g.startGame(g.id)
			return r, cmd
		}
	case timer.TickMsg:
		if msg.Timeout {
			g.finished = true
			break
		}

		if g.started && !g.finished && g.mode == "time" {
			g.sample()
		}
	case GameTickMsg:
		if g.mode != "words" {
			break
		}

		if msg.gameId != g.id {
			break
		}

		if !g.finished && msg.Timeout {
			g.finished = true
			break
		}

		if g.started && !g.finished {
			g.sample()
			return r, g.tickCmd(false, g.id)
		}
	}

	var timerCmd tea.Cmd

	if g.started && !g.finished && g.mode == "time" {
		g.timer, timerCmd = g.timer.Update(msg)
	}

	if g.finished {
		//r.stats.TotalCompleted++
		//r.stats.LastTestId++

		pairs := g.computeMismatchedWords()

		words := make([]string, 0, len(pairs))

		for _, pair := range pairs {
			words = append(words, pair.word)
		}

	   	g.missedWords = strings.Join(words, " ")

		test := &RacerTest{
			Accuracy: g.accuracy*100,
			Target: g.target,
			Input: string(g.inputs),
			Test: g.testName,
			Time: g.testDuration,
			Mode: g.mode,
			TestSize: g.wordsTestSize,
			AllowBackspace: g.allowBackspace,
			Cps: computeCps(g.charsPerSec),
			Wpm: 0,
			Rle: g.alignment.rle(),
			RawInput: g.alignment.rawString(),
			SampleRate: 1,
			//AccList: slices.Clone(g.accs),
			//CpsList: slices.Clone(g.charsPerSec),
			//WpmList: slices.Repeat([]int{0}, len(g.charsPerSec)), 
		}

		//stats := r.stats.Copy()

		//req := saveGameStatsAndTestRequest{
		//	stats: stats,
		//	test: test,
		//}

		var cmd tea.Cmd

		if g.id % 3 == 0 {
			cmd = r.ProcessTestsCmd()
		}

		return r, tea.Batch(cmd, timerCmd, r.insertRacerTestCmd(test), cmd)
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
			cmd = g.startGame(g.id)
		}
	}

	var timerCmd tea.Cmd

	if g.started && !g.finished && g.mode == "time" {
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
			cmd = g.startGame(g.id)
		}
	}

	var timerCmd tea.Cmd
	var saveCmd tea.Cmd

	if g.started && !g.finished && g.mode == "time" {
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
			settings.Next()
			for settings.IsHidden() {
				settings.Next()
			}
		case "k":
			settings.Prev()
			for settings.IsHidden() {
				settings.Prev()
			}
		case "h":
			settings.PrevSettingsOption()
		case "l":
			settings.NextSettingsOption()
		case "enter":
			settings.SelectSettingsOption()
			optionName, value := settings.GetCurrentSelectedOptionPair()

			if optionName != "mode" && settings.showSave {
				return r, settings.SaveSettings
			}

			switch value {
			case "time":
				settings.HideSettingsOption("words test size")
				settings.UnhideSettingsOption("time")
			case "words":
				settings.HideSettingsOption("time")
				settings.UnhideSettingsOption("words test size")
			}

			if settings.showSave {
				return r, settings.SaveSettings
			}
		//case "s":
		//	if settings.showSave {
		//		return r, settings.SaveSettings
		//	}
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
		tests, err := GetAllTests(r.getAllTestsStmt)

		if err != nil {
			return getAllTestsErr(err)
		}

		return getAllTestsSuccess{
			tests: tests,
		}
	}
}

func (t *RacerTest) row() []string {
	return []string{
		fmt.Sprintf("%d", t.Id),
		t.Test,
		fmt.Sprintf("%d", t.Time),
		t.Mode,
		fmt.Sprintf("%v", t.AllowBackspace),
		fmt.Sprintf("%d", t.TestSize),
		fmt.Sprintf("%.2f", t.Accuracy),
		t.Target,
		t.Input,
		fmt.Sprintf("%d", t.Wpm),
		fmt.Sprintf("%d", t.Cps),
		t.Rle,
		fmt.Sprintf("%s", t.RawInput),
	}
}

func convertTestsToRows(tests []*RacerTest) []table.Row {
	rows := make([]table.Row, 0, len(tests))

	for _, test := range tests {
		rows = append(rows, test.row())
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

type chunkTickMsg struct{}

func doChunkTick() tea.Cmd {
	return tea.Tick(10*time.Millisecond, func(_ time.Time) tea.Msg {
		return chunkTickMsg{}
	})
}

type chunkCharTickMsg byte

func doChunkTick2(chunk string, idx int) tea.Cmd {
	return tea.Tick(10*time.Millisecond, func(_ time.Time) tea.Msg {
		return chunkCharTickMsg(chunk[idx])
	})
}

func (r *RacerModel) updateIntroModel(msg tea.Msg) (tea.Model, tea.Cmd) {
	intro := r.introModel

	if intro.done {
		r.SetState(PLAYER_INFO)
		return r, nil
	}

	switch msg := msg.(type) {
	case chunkCharTickMsg:
		c := msg
		intro.out = append(intro.out, byte(c))
		if intro.idx + 1 < len(intro.lines[intro.lineIdx]) {
			intro.idx++
			return r, doChunkTick2(string(intro.lines[intro.lineIdx]), intro.idx)
		}
	 case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			intro.done = true
			r.SetState(PLAYER_INFO)
			return r, nil
		case "esc":
			r.SetState(MAIN_MENU)
			intro.reset()
			return r, nil
		default:
			if intro.lineIdx + 1 < len(intro.lines) {
				intro.waitForInput = false
				intro.out = []byte{}
				intro.lineIdx++
				intro.idx = 0
				return r, doChunkTick2(string(intro.lines[intro.lineIdx]), intro.idx)
			} else {
				intro.done = true
			}
			return r, doChunkTick()
		}
	}
	return r, nil
}

func (r *RacerModel) updatePlayerInfoModel(msg tea.Msg) (tea.Model, tea.Cmd) {
	info := r.playerInfoModel

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			info.input = []byte{}
			r.SetState(MAIN_MENU)
			return r, nil
		case tea.KeyRunes:
			char := byte(msg.Runes[0])
			if isValidChar(char) {
				info.input = append(info.input, char)
			}
		case tea.KeyBackspace:
			if len(info.input) != 0 {
				info.input = info.input[:len(info.input)-1]
			}
		case tea.KeySpace:
			info.input = append(info.input, ' ')
		case tea.KeyEnter:
			name := string(info.input)
			info.value = name
			if len(name) == 0 {
				break
			}
			params := &PlayerInfoInsertParams{
				name: name,
			}
			return r, r.insertPlayerInfoCmd(params)
		}
	case insertPlayerSuccess:
		info.found = true
		r.playerFound = true
		r.playerInfo = &PlayerInfo{
			name: info.value,
		}
		r.playerFound = true
	}
	return r, nil
}

type insertPlayerInfoErr error
type insertPlayerSuccess struct{}

func (r *RacerModel) insertPlayerInfoCmd(params *PlayerInfoInsertParams) tea.Cmd {
	return func() tea.Msg {
		if err := InsertPlayerInfo(r.db, params); err != nil {
			return insertPlayerInfoErr(err)
		}

		return insertPlayerSuccess{}
	}
}

type processTestsErr error

type wordPair struct {
	word string
	count int
}

func createWordPairs(wordCount map[string]int) []wordPair {
	pairs := make([]wordPair, 0, len(wordCount))

	for key, value := range wordCount {
		pairs = append(pairs, wordPair{ key, value })
	}

	slices.SortStableFunc(pairs, func(a, b wordPair) int {
		if a.count == b.count {
			return 0
		} else if a.count > b.count {
			return 1
		} else {
			return -1
		}
	})

	return pairs
}

//func filter[S []T, T any](s S, f func(T) bool) S {
//	idx := 0
//	for _, item := range s {
//		if f(item) {
//			s[idx] = item
//			idx++
//		}
//	}
//	s = s[:idx]
//	return s
//}

type RefreshModel struct{}

func (r *RacerModel) ProcessTestsCmd() tea.Cmd {
	return func() tea.Msg {
		tests, err := GetAllTests(r.getAllTestsStmt)

		if err != nil {
			return processTestsErr(err)
		}

		wordCount := make(map[string]int)

		for _, test := range tests {
			input := test.Input
			target := test.Target[:len(input)]
			leftIdx := 0
			for i := 0; i < len(target); i++ {
				if target[i] != ' ' {
					continue
				}

				if string(input[leftIdx:i]) != string(target[leftIdx:i]) {
					wordCount[string(target[leftIdx:i])]++
				}

				i++
				leftIdx = i
			}

			if string(input[leftIdx:]) != string(target[leftIdx:]) {
				wordCount[string(target[leftIdx:])]++
			}
		}

		pairs := createWordPairs(wordCount)

		if len(pairs) > 50 {
			pairs = pairs[:50]
		}

		words := make([]string, 0, len(pairs))

		for _, pair := range pairs {
			words = append(words, pair.word)
		}

		wordList := &WordList{
			Name: "frequent",
			Words: words,
		}

		if err := wordList.Save(); err != nil {
			return processTestsErr(err)
		}

		return UpdateWordDb{
			l: wordList,
		}

	}
}
