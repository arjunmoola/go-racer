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
	//"slices"
)

var rpcg = rand.New(rand.NewPCG(0,1))

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("200"))
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("200"))
	charMatchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#008000"))
	charMismatchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))
	highlightStyle = lipgloss.NewStyle().Background(lipgloss.Color("32"))
	viewStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).Align(lipgloss.Left).Height(3)
	overlapSpaceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A9A9A9")).Underline(true)
	timerStyle = lipgloss.NewStyle().PaddingRight(3)
)

type editOp interface {
	Byte() byte
	String() string
	Code() string
}

type matchOp byte

func (op matchOp) String() string {
	return fmt.Sprintf("m(%s)", string(op))
}

func (op matchOp) Byte() byte {
	return byte(op)
}

func (op matchOp) Code() string {
	return "m"
}

type mismatchOp byte

func (op mismatchOp) String() string {
	return fmt.Sprintf("s(%s)", string(op))
}

func (op mismatchOp) Byte() byte {
	return byte(op)
}

func (op mismatchOp) Code() string {
	return "s"
}

type deleteOp byte

func (op deleteOp) String() string {
	return fmt.Sprintf("d(%s)", string(op))
}

func (op deleteOp) Byte() byte {
	return byte(op)
}

func (op deleteOp) Code() string {
	return "d"
}

type alignment []editOp

func (a alignment) String() string {
	builder := &strings.Builder{}

	for _, op := range a {
		builder.WriteString(op.String())
	}

	return builder.String()
}

func (a alignment) rle() string {
	builder := &strings.Builder{}

	if len(a) == 0 {
		return ""
	}

	if len(a) == 1 {
		return a[0].Code()
	}

	matchLen := 1
	prevOp := a[0]

	for i := 1; i < len(a); i++ {
		op := a[i]

		if op.Code() == prevOp.Code() {
			matchLen++
		} else {
			if matchLen == 1 {
				fmt.Fprintf(builder, "%s", prevOp.Code())
			} else {
				fmt.Fprintf(builder, "%d%s", matchLen, prevOp.Code())
				matchLen = 1
			}
		}
		prevOp = op
	}

	if matchLen == 1 {
		fmt.Fprintf(builder, "%s", prevOp.Code())
	}

	return builder.String()
}


type Game struct {
	id int
	testName string
	testDuration int
	testSize int
	wordsTestSize int

	racer *RacerModel
	mode string
	debug bool
	started bool
	target string
	inputs []byte
	alignment alignment
	charIdx int
	idx int
	timer timer.Model
	finished bool

	ticks int

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

	numMatches int
	numMisses int
	numCharsPerSec int
	charsPerSec []int
	charBuffer []byte

	accuracy float64
	accs []float64

	wordIdx int
	curWord string

	allowBackspace bool
	curWpm int
	sampleIdx int
	prevSampleIdx int
	samples []string
}

func NewGameFromConfig(config *Config) *Game {
	game := &Game{
		numWordsPerLine: config.NumWordsPerLine,
		windowSize: config.WindowSize,
		testSize: config.TestSize,
		debug: config.Debug,
	}
	return game
}

func (g *Game) createTest() {
	g.id++
	racer := g.racer
	config := racer.config

	g.testName = config.Words
	g.testDuration = config.Time
	g.testSize = config.TestSize
	g.wordsTestSize = config.WordsTestSize
	g.allowBackspace = config.AllowBackspace
	g.mode = config.GameMode

	g.timer = timer.New(time.Duration(g.testDuration)*time.Second)

	selectedWordList, _ := racer.wordDb.Get(g.testName)
	words := selectedWordList.Words

	n := len(words)

	var testSize int

	if g.mode == "words" {
		testSize = g.wordsTestSize
	} else {
		testSize = g.testSize
	}

	test := make([]string, 0, testSize)

	for range testSize {
		idx := rand.IntN(n)
		test = append(test, words[idx])
	}

	target := strings.Join(test, " ")

	lineOffsets := append(make([]int, 0), 0)
	count := 0

	for i := 0; i < len(target); i++ {
		if target[i] == ' ' {
			count++
			if count == g.numWordsPerLine {
				count = 0
				lineOffsets = append(lineOffsets, i+1)
			}
		}
	}

	g.lineOffsets = lineOffsets
	g.curLine = 0
	g.curWindow = 0
	g.leftIdx = 0
	g.sampleIdx = 0
	g.prevSampleIdx = 0

	if g.curWindow + g.windowSize < len(g.lineOffsets) {
		g.rightIdx = g.lineOffsets[g.curWindow+g.windowSize]
	} else {
		g.rightIdx = len(target)
	}

	g.target = target
}

func (g *Game) Reset() {
	g.numMatches = 0
	g.numMisses = 0
	g.accuracy = 0
	g.numCharsPerSec = 0
	g.charsPerSec = []int{}
	g.accs = []float64{}
	g.charBuffer = []byte{}
	g.alignment = nil
	g.inputs = nil
	g.charIdx = 0
	g.idx = 0
	g.timer = timer.New(time.Second*30)
	g.ticks = 0
	g.finished = false
	g.started = false
}

func (g *Game) updateSampleIdx() {
	g.prevSampleIdx = g.sampleIdx
	g.sampleIdx = g.idx
}

func (g *Game) sample() {
	if g.idx >= g.prevSampleIdx {
		s := string(g.target[g.prevSampleIdx:g.idx])
		g.samples = append(g.samples, s)
		g.updateSampleIdx()

	}
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

		if g.curLine-1 > -1 && g.idx >= g.lineOffsets[g.curLine-1] && g.idx < g.lineOffsets[g.curLine] {
			g.curLine--
			g.curWindow = (g.curLine/g.windowSize)*g.windowSize

			if g.curLine % g.windowSize == 2 {
				g.rightIdx = g.leftIdx
				g.leftIdx = g.lineOffsets[g.curWindow]
			}

		}
	}
}

func isValidChar(char byte) bool {
	return 'a' <= char && char <= 'z' || 'A' <= char && char <= 'Z' || '0' <= char && char <= '9' || char == ' '
}

func (g *Game) appendByte(char byte) {
	g.inputs = append(g.inputs, char)
}

func (g *Game) appendOp(op editOp) {
	g.alignment = append(g.alignment, op)
}

func (g *Game) trimByte() byte {
	b := g.inputs[len(g.inputs)-1]
	g.inputs = g.inputs[:len(g.inputs)-1]
	return b
}

type GameTickMsg struct{
	gameId int
	Timeout bool
}

func (g *Game) tickCmd(finished bool, id int) tea.Cmd {
	return tea.Tick(1*time.Second, func(_ time.Time) tea.Msg {
		return GameTickMsg{
			gameId: id,
			Timeout: finished,
		}
	})
}

func (g *Game) startGame(id int) tea.Cmd {
	if g.mode == "words" {
		return g.tickCmd(false, id)
	}
	return g.timer.Init()
}

func (g *Game) stopGame(id int) tea.Cmd {
	if g.mode == "words" {
		return g.tickCmd(true, id)
	}
	return g.timer.Stop()
}

func computeCps(nums []int) (total int) {
	if len(nums) == 0 {
		return total
	}

	for _, num := range nums {
		total += num
	}

	return total/len(nums)
}

func (g *Game) View() string {
	builder := &strings.Builder{}

	if g.finished {
		builder.WriteString("Results: \n\n")
		fmt.Fprintf(builder, "name: %s\n", g.testName)
		fmt.Fprintf(builder, "mode: %s\n", g.mode)
		fmt.Fprintf(builder, "time: %d s\n", g.ticks)
		fmt.Fprintf(builder, "accuracry: %.2f%%\n", g.accuracy*100)
		fmt.Fprintf(builder, "cps: %d\n", computeCps(g.charsPerSec))
		fmt.Fprintf(builder, "%v\n", g.charsPerSec)
		fmt.Fprintf(builder, "%s\n", g.alignment)
		fmt.Fprintf(builder, "rle: %s\n", g.alignment.rle())
		builder.WriteRune('\n')
		fmt.Fprintf(builder, "press esc to go to main menu\n")
		fmt.Fprintf(builder, "press r to restart\n")
		fmt.Fprintf(builder, "press enter to go to next test\n")
		//builder.WriteString("press ctrl+c to quit\n")
		return builder.String()
	}

	if !g.started {
		builder.WriteString("press enter to start\n")
		builder.WriteString("press esc to go to main menu\n")
	} else {
		fmt.Fprintf(builder, "name: %s\n", g.testName)
		timeView := ""
		switch g.mode {
		case "time":
			timeView = timerStyle.Render(g.timer.View())
		case "words":
			timeView = timerStyle.Render(fmt.Sprintf("%d", g.ticks))
		}

		var acc float64

		if len(g.accs) == 0 {
			acc = g.accuracy
		} else {
			acc = g.accs[len(g.accs)-1]
		}
		acc *= 100
		accView := timerStyle.Render(fmt.Sprintf("accuracry: %.2f %%", acc))
		wpmView := timerStyle.Render(fmt.Sprintf("wpm: %d", g.curWpm))
		modeView := timerStyle.Render(fmt.Sprintf("mode: %s", g.mode))
		var cps int
		if n := len(g.charsPerSec)-1; n >= 1 {
			cps = g.charsPerSec[n]
		}
		cpsView := timerStyle.Render(fmt.Sprintf("cps: %d:", cps))
		builder.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, timeView, modeView, accView, wpmView, cpsView))

		//builder.WriteString(g.timer.View())
		builder.WriteRune('\n')

		s := g.render()
		builder.WriteString(viewStyle.Render(s))
		builder.WriteRune('\n')

		if g.debug {
			fmt.Fprintf(builder, "ids: %d\n", g.idx)
			fmt.Fprintf(builder, "currentLine: %d\n", g.curLine)
			fmt.Fprintf(builder, "currentLineOffset: %d\n", g.lineOffsets[g.curLine])
			fmt.Fprintf(builder, "currentWindow: %d\n", g.curWindow)
			fmt.Fprintf(builder, "leftIdx: %d rightIdx: %d\n", g.leftIdx, g.rightIdx)
			fmt.Fprintf(builder, "mod: %d\n", g.curLine % g.windowSize)
			fmt.Fprintf(builder, "number of windows: %d\n", len(g.lineOffsets)/3)
			fmt.Fprintf(builder, "number of lines: %d\n", len(g.lineOffsets))
			builder.WriteString(renderLineOffsets(g.lineOffsets, g.curLine, g.curWindow, g.windowSize))
		}
		//fmt.Fprintf(builder, "lineOffsets: %v\n", renderLineOffsets(g.lineOffsets, g.curLine, g.curWindow, g.windowSize))
	}

	builder.WriteString("\n\n")

	builder.WriteString("press ctrl+c to quit\n")
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
		} else if g.inputs[i] == ' ' && g.target[i] != ' ' {
			s += overlapSpaceStyle.Render(string(g.target[i]))
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
