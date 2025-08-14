package racer

import (
	tea "github.com/charmbracelet/bubbletea"
	"slices"
	"strings"
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
