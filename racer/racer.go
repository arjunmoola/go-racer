package racer

import (
	"os"
	"fmt"
	"slices"
	tea "github.com/charmbracelet/bubbletea"
	//"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/bubbles/timer"
	"strings"
	"errors"
	"time"
	"math/rand/v2"
)

var rpcg = rand.New(rand.NewPCG(0,1))

var (
	ErrModelNotFound = errors.New("model not found")
)

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("200"))
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("100"))
	charMatchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#008000"))
	charMismatchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))
	highlightStyle = lipgloss.NewStyle().Background(lipgloss.Color("32"))
)

type List struct {
	items []string
	cursor int
	selectedValue string
	selectedIdx int
}

func NewList() *List {
	return &List{
		selectedIdx: -1,
	}
}

func (l *List) SetItems(items []string) {
	l.items = slices.Clone(items)
	l.cursor = 0
	l.selectedValue = ""
	l.selectedIdx = -1
}

func (l *List) Next() {
	if l.cursor + 1 < len(l.items) {
		l.cursor++
	}
}

func (l *List) Prev() {
	if l.cursor - 1 > -1 {
		l.cursor--
	}
}

func (l *List) SetSelection() {
	l.selectedValue = l.items[l.cursor]
	l.selectedIdx = l.cursor
}

func (l *List) RemoveSelection() {
	l.selectedValue = ""
	l.selectedIdx = -1
}

func (l *List) SelectedValue() string {
	return l.selectedValue
}

func (l *List) Init() tea.Cmd {
	return nil
}

func (l *List) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j":
			l.Next()
		case "k":
			l.Prev()
		}
	}

	return l, nil
}

func  (l *List) View() string {
	builder := &strings.Builder{}
	for idx, item := range l.items {
		if idx == l.cursor {
			builder.WriteString(cursorStyle.Render(item)+"\n")
		} else {
			builder.WriteString(item+"\n")
		}
	}
	//builder.WriteString("press q to exit\n")
	return builder.String()
}

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

func (m *Menu) Init() tea.Cmd {
	return nil
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

type Game struct {
	started bool
	target string
	inputs []byte
	charIdx int
	idx int
	timer timer.Model
	finished bool
	ticks int
	testSize int
	numWordsPerLine int
	lineOffsets []int

	wordDb *WordDb
	defaultWordList *WordList
}

func NewGame() *Game {
	t := timer.New(time.Second*30)

	return &Game{
		timer: t,
	}
}

func (g *Game) createTest() string {
	words := g.defaultWordList.Words
	n := len(words)


	test := make([]string, 0, g.testSize)

	for range g.testSize {
		idx := rand.IntN(n)
		test = append(test, words[idx])
	}

	return strings.Join(test, " ")
}

func (g *Game) SetWordDb(wordDb *WordDb) {
	g.wordDb = wordDb
}

func (g *Game) SetDefaultWordList(name string) {
	g.defaultWordList = g.wordDb.wordLists[name]
}

func (g *Game) Reset() {
	g.inputs = nil
	g.charIdx = 0
	g.idx = 0
	g.timer = timer.New(time.Second*30)
	g.ticks = 0
	g.finished = false
	g.started = false
}

func (g *Game) SetTarget(target string) {
	g.target = target
}

func (g *Game) Init() tea.Cmd {
	return nil
}

func (g *Game) incIndex() {
	if g.idx+1 < len(g.target) {
		g.idx++
	}
}

func (g *Game) decIndex() {
	if g.idx-1 > -1 {
		g.idx--
	}
}

func isValidChar(char byte) bool {
	return 'a' <= char && char <= 'z' || 'A' <= char && char <= 'Z' || '0' <= char && char <= '9' || char == ' '
}

func (g *Game) appendByte(char byte) {
	if isValidChar(char) {
		g.inputs = append(g.inputs, char)
	}
}

func (g *Game) trimByte() {
	g.inputs = g.inputs[:len(g.inputs)-1]
}

func (g *Game) startGame() tea.Cmd {
	return g.timer.Init()
}

func (g *Game) stopGame() tea.Cmd {
	return g.timer.Stop()
}

func (g *Game) gotoMainMenu() tea.Msg {
	return mainMenuEvent{}
}

func (g *Game) gotoStartGame() tea.Msg {
	return startGameEvent{}
}

func (g *Game) updateGameFinished(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "b":
			g.Reset()
			return g, g.gotoMainMenu
		case "esc":
			return g, tea.Quit
		case "r":
			g.Reset()
			return g, g.gotoStartGame
		}
	}

	return g, nil
}

func (g *Game) updateGameNotStarted(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "b":
			g.Reset()
			return g, g.gotoMainMenu
		case "esc":
			return g, tea.Quit
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

	return g, tea.Batch(cmd, timerCmd)
}

func (g *Game) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !g.started {
		return g.updateGameNotStarted(msg)
	}

	if g.finished {
		return g.updateGameFinished(msg)
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

	return g, tea.Batch(cmd, timerCmd)
}

func (g *Game) View() string {
	builder := &strings.Builder{}

	if g.finished {
		builder.WriteString("Results: \n\n")
		fmt.Fprintf(builder, "time: %d s\n", g.ticks)
		fmt.Fprintf(builder, "press b to go to main menu\n")
		fmt.Fprintf(builder, "press r to restart\n")
		builder.WriteString("press esc to quit\n")
		return builder.String()
	}

	if !g.started {
		builder.WriteString("press enter to start\n")
		builder.WriteString("press b to go to main menu\n")
	} else {
		fmt.Fprintf(builder, "name: %s\n", g.defaultWordList.Name)
		builder.WriteString(g.timer.View())
		builder.WriteRune('\n')

		end := min(len(g.target), len(g.inputs))
		for i := range end {
			if g.target[i] == g.inputs[i] {
				builder.WriteString(charMatchStyle.Render(string(g.inputs[i])))
			} else {
				builder.WriteString(charMismatchStyle.Render(string(g.inputs[i])))
			}
		}

		builder.WriteString(highlightStyle.Render(string(g.target[end])))

		if end+1 < len(g.target) {
			builder.WriteString(g.target[end+1:])
		}

	}

	builder.WriteString("\n\n")

	builder.WriteString("press esc to quit\n")
	return builder.String()
}

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
	if _, err := tea.NewProgram(r).Run(); err != nil {
		return err
	}
	return nil
}
