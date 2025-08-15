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
	"slices"
)

var rpcg = rand.New(rand.NewPCG(0,1))

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("200"))
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("100"))
	charMatchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#008000"))
	charMismatchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))
	highlightStyle = lipgloss.NewStyle().Background(lipgloss.Color("32"))
	viewStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).Align(lipgloss.Left).Height(3)
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
	curLine int
	windowSize int
	windowOffsets []int
	curWindow int
	leftIdx int
	leftLineIdx int
	rightIdx int
	rightLineIdx int
	bsearchIdx int

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
		dur = 120
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

	target := strings.Join(test, " ")

	lineOffsets := append(make([]int, 0), 0)
	count := 0

	for i := 0; i < len(target); i++ {
		if target[i] == ' ' {
			count++
			if count == 15 {
				count = 0
				lineOffsets = append(lineOffsets, i+1)
			}
		}
	}

	g.lineOffsets = lineOffsets
	g.curLine = 0
	g.curWindow = 0
	g.windowSize = 3
	g.leftIdx = 0

	if g.curWindow + g.windowSize < len(g.lineOffsets) {
		g.rightIdx = g.lineOffsets[g.curWindow+g.windowSize]
	} else {
		g.rightIdx = len(target)
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

		if g.curLine+1 < len(g.lineOffsets) && g.idx == g.lineOffsets[g.curLine+1] {
			g.curLine++
			g.curWindow = (g.curLine/g.windowSize)*g.windowSize
			if g.curLine % g.windowSize == 0 {
				g.leftIdx = g.rightIdx
				if g.curWindow+g.windowSize < len(g.target) {
					g.rightIdx = g.lineOffsets[g.curWindow+g.windowSize]
				} else {
					g.rightIdx = len(g.target)
				}
			}
		}
	}
}

func (g *Game) decIndex() {
	if g.idx-1 > -1 {
		g.idx--

		idxLine, _ := slices.BinarySearch(g.lineOffsets, g.idx)
		g.bsearchIdx = idxLine

		if g.curLine-1 > -1 && idxLine == g.curLine-1 {
			g.curLine--
			g.curWindow = (g.curLine/g.windowSize)*g.windowSize

			if g.curLine % g.windowSize == 2 {
				g.rightIdx = g.leftIdx
				if g.curWindow-g.windowSize > -1 {
					g.leftIdx = g.lineOffsets[g.curWindow-g.windowSize]
				} else {
					g.leftIdx = 0
				}
			}

		}
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

		s := g.render()

		builder.WriteString(viewStyle.Render(s))
		builder.WriteRune('\n')
		fmt.Fprintf(builder, "ids: %d\n", g.idx)
		fmt.Fprintf(builder, "currentLine: %d\n", g.curLine)
		fmt.Fprintf(builder, "currentLineOffset: %d\n", g.lineOffsets[g.curLine])
		fmt.Fprintf(builder, "bsearchIdx: %d\n", g.bsearchIdx)
		fmt.Fprintf(builder, "currentWindow: %d\n", g.curWindow)
		fmt.Fprintf(builder, "leftIdx: %d rightIdx: %d\n", g.leftIdx, g.rightIdx)
		fmt.Fprintf(builder, "mod: %d\n", g.curLine % g.windowSize)
		fmt.Fprintf(builder, "number of windows: %d\n", len(g.lineOffsets)/3)
		fmt.Fprintf(builder, "number of lines: %d\n", len(g.lineOffsets))
		builder.WriteString(renderLineOffsets(g.lineOffsets, g.curLine, g.curWindow, g.windowSize))
		//fmt.Fprintf(builder, "lineOffsets: %v\n", renderLineOffsets(g.lineOffsets, g.curLine, g.curWindow, g.windowSize))
	}

	builder.WriteString("\n\n")

	builder.WriteString("press esc to quit\n")
	return builder.String()
}

var (
	defaultStyle = lipgloss.NewStyle()
	lineOffsetCursorStyle = defaultStyle.Foreground(lipgloss.Color("200"))
	windowStyle = defaultStyle.BorderStyle(lipgloss.NormalBorder()).Height(1)
	underlineStyle = defaultStyle.Underline(true).UnderlineSpaces(true)
)

func renderLineOffsets(lineOffsets []int, curIndex int, windowIdx int, windowSize int) string {
	builder := &strings.Builder{}

	for i := range windowIdx {
		num := strconv.Itoa(lineOffsets[i])
		builder.WriteString(defaultStyle.Render(num))
		builder.WriteString(" ")
	}

	leftStr := builder.String()
	builder.Reset()

	windowStr := ""

	for i := windowIdx; i < len(lineOffsets) && i < windowIdx+windowSize; i++ {
		num := strconv.Itoa(lineOffsets[i])
		if i == curIndex {
			windowStr += lineOffsetCursorStyle.Render(num)
		} else {
			windowStr += defaultStyle.Render(num)
		}
		windowStr += " "
	}

	windowStr = windowStyle.Render(windowStr)

	for i := windowIdx+windowSize; i < len(lineOffsets); i++ {
		num := strconv.Itoa(lineOffsets[i])
		builder.WriteString(defaultStyle.Render(num))
		builder.WriteString(" ")
	}

	rightStr := builder.String()
	builder.Reset()

	return lipgloss.JoinHorizontal(lipgloss.Center, leftStr, windowStr, rightStr)
}

func (g *Game) render() string {
	lineOffsets := g.lineOffsets
	leftIdx := g.leftIdx
	rightIdx := g.rightIdx
	windowIdx := g.curWindow

	lineIdx := windowIdx+1
	end := g.idx

	var s string
	for i := leftIdx; i < end; i++ {
		if g.target[i] == g.inputs[i] {
			if g.lineOffsets[lineIdx] == i+1 && g.target[i] == ' ' {
				s += charMatchStyle.Render("\n")
				if lineIdx+1 < len(lineOffsets) {
					lineIdx++
				}
			} else{
				s += charMatchStyle.Render(string(g.inputs[i]))
			}
			//builder.WriteString(charMatchStyle.Render(string(g.inputs[i])))
		} else {
			s += charMismatchStyle.Render(string(g.inputs[i]))
			//builder.WriteString(charMismatchStyle.Render(string(g.inputs[i])))
		}
	}

	s += highlightStyle.Render(string(g.target[end]))

	if end == lineOffsets[lineIdx]-1 {
		s += "\n"
		if lineIdx+1 < len(lineOffsets) {
			lineIdx++
		}
	}

	//builder.WriteString(highlightStyle.Render(string(g.target[end])))

	for i := end+1; i < rightIdx; i++ {
		if i+1 == lineOffsets[lineIdx] {
			s += "\n"
			if lineIdx+1 < len(lineOffsets) {
				lineIdx++
			}
		} else {
			s += string(g.target[i])
		}

	}

	return s
}
