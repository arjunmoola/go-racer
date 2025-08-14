package racer

import (
	"github.com/charmbracelet/bubbles/timer"
	"math/rand/v2"
	"strings"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"time"
	"fmt"
	"strconv"
)

var rpcg = rand.New(rand.NewPCG(0,1))

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("200"))
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("100"))
	charMatchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#008000"))
	charMismatchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))
	highlightStyle = lipgloss.NewStyle().Background(lipgloss.Color("32"))
	viewStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).Align(lipgloss.Center)
)

type Game struct {
	racer *RacerModel
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
	selectedTest := g.racer.settings.selectedOptions["words"]

	var words []string

	if selectedTest == "" {
		words = g.defaultWordList.Words
	} else {
		g.defaultWordList = g.wordDb.wordLists[selectedTest]
		words = g.wordDb.wordLists[selectedTest].Words
	}

	selectedTime := g.racer.settings.selectedOptions["time"]

	var dur int64

	if selectedTime == "" {
		dur = 30
	} else {
		dur, _ = strconv.ParseInt(selectedTime, 10, 64)
	}

	g.timer = timer.New(time.Duration(dur)*time.Second)
 
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


func (g *Game) updateGameFinished(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "b":
			g.Reset()
			g.racer.SetState(MAIN_MENU)
		case "esc":
			return g, tea.Quit
		case "r":
			g.Reset()
			g.racer.SetState(GAME)
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
			g.racer.SetState(MAIN_MENU)
			return g, nil
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

		var s string
		for i := range end {
			if g.target[i] == g.inputs[i] {
				s += charMatchStyle.Render(string(g.inputs[i]))
				//builder.WriteString(charMatchStyle.Render(string(g.inputs[i])))
			} else {
				s += charMismatchStyle.Render(string(g.inputs[i]))
				//builder.WriteString(charMismatchStyle.Render(string(g.inputs[i])))
			}
		}

		s += highlightStyle.Render(string(g.target[end]))

		//builder.WriteString(highlightStyle.Render(string(g.target[end])))

		if end+1 < len(g.target) {
			//builder.WriteString(g.target[end+1:])
			s += string(g.target[end+1:])
		}

		builder.WriteString(viewStyle.Render(s))

	}

	builder.WriteString("\n\n")

	builder.WriteString("press esc to quit\n")
	return builder.String()
}

func (g *Game) updateSettings(msg tea.Msg) (tea.Model, tea.Cmd) {
	return nil, nil
}
