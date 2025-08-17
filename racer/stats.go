package racer

import (
	tea "github.com/charmbracelet/bubbletea"
	"io"
	"encoding/json"
	"os"
	"path/filepath"
	"errors"
	"fmt"
)

type GameStats struct {
	Total int `json:"total"`
	TotalCompleted int `json:"totalCompleted"`
	TotalAttempted int `json:"totalAttempted"`
	LastTestId int `json:"lastTestId"`
}

type RacerTest struct {
	Id int `json:"-"`
	Test string `json:"test"`
	Time int `json:"time"`
	Target string `json:"target"`
	Input string `json:"input"`
}

var (
	defaultStatsPath = filepath.Join(defaultConfigDir, "stats.json")
	defaultTestDir = filepath.Join(defaultConfigDir, "tests")
)

func ReadGameStats() (*GameStats, error) {
	file, err := os.Open(defaultStatsPath)

	stats := &GameStats{}

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return stats, nil
		}
		return nil, err
	}

	defer file.Close()

	if err := unmarshalGameStats(file, stats); err != nil {
		return nil, err
	}

	return stats, nil
}

func unmarshalGameStats(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

func (s *GameStats) writeTo(w io.Writer) error {
	return json.NewEncoder(w).Encode(s)
}

func (s *GameStats) Save() error {
	file, err := os.Create(defaultStatsPath)

	if err != nil {
		return err
	}

	defer file.Close()

	return s.writeTo(file)
}

func (s *GameStats) Copy() *GameStats {
	stats := &GameStats{
		Total: s.Total,
		TotalCompleted: s.TotalCompleted,
		TotalAttempted: s.TotalAttempted,
		LastTestId: s.LastTestId,
	}
	return stats
}

func (t *RacerTest) writeTo(w io.Writer) error {
	return json.NewEncoder(w).Encode(t)
}

func (t *RacerTest) Save() error {

	_, err := os.Lstat(defaultTestDir)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(defaultTestDir, 0777); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	testPath := filepath.Join(defaultTestDir, fmt.Sprintf("%d.json", t.Id))

	file, err := os.Create(testPath)

	if err != nil {
		return err
	}

	defer file.Close()

	return t.writeTo(file)
}

type SaveRacerTestErr error
type SaveGameStatsErr error
type SaveStatsSuccess struct{}
type SaveRacerTestSuccess struct{}
type SaveStatsAndTestErr error
type SaveStatsAndTestSuccess struct{}

func SaveStats(s *GameStats) tea.Cmd {
	return func() tea.Msg {
		if err := s.Save(); err != nil {
			return SaveGameStatsErr(err)
		}
		return SaveStatsSuccess{}
	}
}

func SaveRacerTest(t *RacerTest) tea.Msg {
	_, err := os.Lstat(defaultTestDir)

	testPath := filepath.Join(defaultTestDir, fmt.Sprintf("%d.json", t.Id))

	file, err := os.Create(testPath)

	if err != nil {
		return SaveRacerTestErr(err)
	}

	defer file.Close()

	if err := t.writeTo(file); err != nil {
		return SaveRacerTestErr(err)
	}

	return SaveRacerTestSuccess{}
}
