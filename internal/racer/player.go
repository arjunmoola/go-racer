package racer

import (
	"strings"
	"github.com/charmbracelet/lipgloss"
	"fmt"
)

var (
	inputStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder())
)

type PlayerInfoModel struct {
	found bool
	input []byte
	value string
	idx int
}

func NewPlayerInfoModel() *PlayerInfoModel {
	return &PlayerInfoModel{}
}


func (m *PlayerInfoModel) render() string {
	builder := &strings.Builder{}

	if m.found {
		fmt.Fprintf(builder, "Welcome: %s\n", m.value)
	} else {
		builder.WriteString("Enter Name:\n")
		builder.WriteString(inputStyle.Render(string(m.input)))
		builder.WriteRune('\n')
	}

	builder.WriteString("press esc to go back to main menu\n")
	builder.WriteString("press ctrl+c to exit\n")
	return builder.String()
}
