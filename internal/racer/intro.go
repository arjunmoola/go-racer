package racer

type IntroModel struct {
	lines []string
	out []byte
	idx int
	lineIdx int
	done bool
	waitForInput bool
}

func NewIntroModel(text []string) *IntroModel {
	return &IntroModel{
		lines: text,
	}
}

func (m *IntroModel) nextChar() bool {
	m.idx++
	if m.idx == len(m.lines[m.lineIdx]) {
		m.idx = 0
		m.lineIdx++
		if m.lineIdx == len(m.lines) {
			m.lineIdx--
			m.idx = len(m.lines[m.lineIdx])-1
			return false
		}
	}
	return true
}

func (m *IntroModel) char() string {
	return string(m.lines[m.lineIdx][m.idx])
}

func (m *IntroModel) appendByte() {
	c := m.lines[m.lineIdx][m.idx]
	m.out = append(m.out, c)
}

func (m *IntroModel) reset() {
	m.idx = 0
	m.lineIdx = 0
	m.done = false
	m.out = []byte{}
}

func (m *IntroModel) render() string {
	return string(m.out)
}
