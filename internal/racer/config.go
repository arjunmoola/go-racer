package racer

import (
	"io"
	"os"
	"errors"
	"encoding/json"
	"path/filepath"
	"github.com/BurntSushi/toml"
)

var (
	ErrInvalidConfig = errors.New("invalid config")
	ErrInvalidConfigKey = errors.New("invalid config key")
)

var (
	defaultConfigDir = os.ExpandEnv("$HOME/.go-racer")
	defaultConfigPath = filepath.Join(defaultConfigDir, "config.json")
	defaultDataDir = filepath.Join(defaultConfigDir, "data")
)

const (
	defaultWindowSize = 3
	defaultGameMode = "time"
	defaultNumWordsPerLine = 20
	defaultAllowBackspace = false
	defaultTestName = "english"
	defaultTestDuration = 30
	defaultTestSize = 500
	defaultWordsTestSize = 25
	defaultMatchColor = "#008000"
	defaultMismatchColor = "#ff0000"
	defaultColor = "#899499"
	defaultLineSpacing = 2
	defaultCursorColor = "32"
	defaultOverlapSpaceColor = "#A9A9A9"
)

type Config struct {
	Words string `json:"words"`
	Time int `json:"time"`
	GameMode string `json:"gameMode"`
	AllowBackspace bool `json:"allowBackspace"`
	Debug bool `json:"debug"`
	WindowSize int `json:"windowSize"`
	NumWordsPerLine int `json:"numWordsPerLine"`
	TestSize int `json:"testSize"`
	WordsTestSize int `json:"wordsTestSize"`
	data string `json:"-"`
}

type Config2 struct {
	Debug bool `toml:"debug"`
	TestName string `toml:"testName"`
	TestDuration int `toml:"testDuration"`
	GameMode string `toml:"gameMode"`
	AllowBackspace bool `toml:"allowBackspace"`
	WindowSize int `toml:"windowSize"`
	NumWordsPerLine int `toml:"numWordsPerLine"`
	TestSize int `toml:"testSize"`
	WordsTestSize int `json:"wordsTestSize"`
	data string `json:"-"`
	MatchColor string `toml:"matchColor"`
	MismatchColor string `toml:"mismatchColor"`
	DefaultColor string `toml:"defaultColor"`
	CursorColor string `toml:"cursorColor"`
	LineSpacing int `toml:"lineSpacing"`
	OverlapSpaceColor string `toml:"overlapSpaceColor"`
}

func getHomeDir() (string, error) {
	return os.UserHomeDir()
}

func unmarshalConfig2(r io.Reader, config any) error {
	decoder := toml.NewDecoder(r)

	if _, err := decoder.Decode(config); err != nil {
		return err
	}

	return nil
}

func (c *Config2) write(w io.Writer) error {
	return toml.NewEncoder(w).Encode(c)
}

func (c *Config2) Save() error {
	file, err := os.Create(filepath.Join(defaultConfigDir, "config.toml"))

	if err != nil {
		return err
	}

	defer file.Close()

	return c.write(file)
}

func ReadConfigFile2() (*Config2, error) {
	configFilePath := filepath.Join(defaultConfigDir, "config.toml")
	
	_, err := os.Lstat(configFilePath)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			config := DefaultConfig2()
			if err := config.Save(); err != nil {
				return nil, err
			}
			return config, nil
		} else {
			return nil, err
		}
	}

	file, err := os.Open(filepath.Join(defaultConfigDir, "config.toml"))

	if err != nil {
		return nil, err
	}

	defer file.Close()

	config := Config2{}

	if err := unmarshalConfig2(file, &config); err != nil {
		return nil, err
	}

	config.data = defaultDataDir

	if config.TestName == "" {
		config.TestName = defaultTestName
	}

	if config.TestDuration <= 0 {
		config.TestDuration = defaultTestDuration
	}

	if config.GameMode == "" {
		config.GameMode = defaultGameMode
	}

	if config.WindowSize <= 0 {
		config.WindowSize = defaultWindowSize
	}

	if config.NumWordsPerLine <= 0 {
		config.NumWordsPerLine = defaultNumWordsPerLine
	}

	if config.TestSize <= 0 {
		config.TestSize = defaultTestSize
	}

	if config.WordsTestSize <= 0 {
		config.WordsTestSize = defaultWordsTestSize
	}

	if config.CursorColor == "" {
		config.CursorColor = defaultCursorColor
	}

	if config.MatchColor == "" {
		config.MatchColor = defaultMatchColor
	}

	if config.MismatchColor == "" {
		config.MismatchColor = defaultMismatchColor
	}

	if config.DefaultColor == "" {
		config.DefaultColor = defaultColor
	}

	if config.LineSpacing == 0 {
		config.LineSpacing = defaultLineSpacing
	}

	if config.OverlapSpaceColor == "" {
		config.OverlapSpaceColor = defaultOverlapSpaceColor
	}

	return &config, nil
}


func ReadConfigFile() (*Config, error) {
	file, err := os.Open(defaultConfigPath)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	config := &Config{}

	if err := unmarshalConfig(file, config); err != nil {
		return nil, err
	}

	config.data = defaultDataDir

	if config.Words == "" {
		config.Words = defaultTestName
	}

	if config.Time <= 0 {
		config.Time = defaultTestDuration
	}

	if config.GameMode == "" {
		config.GameMode = defaultGameMode
	}

	if config.WindowSize <= 0 {
		config.WindowSize = defaultWindowSize
	}

	if config.NumWordsPerLine <= 0 {
		config.NumWordsPerLine = defaultNumWordsPerLine
	}

	if config.TestSize <= 0 {
		config.TestSize = defaultTestSize
	}

	if config.WordsTestSize <= 0 {
		config.WordsTestSize = defaultWordsTestSize
	}

	return config, nil
}

func unmarshalConfig(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

func (c *Config) write(w io.Writer) error {
	return json.NewEncoder(w).Encode(c)
}

func (c *Config) Save() error {
	file, err := os.Create(defaultConfigPath)

	if err != nil {
		return err
	}

	defer file.Close()

	return c.write(file)
}

func DefaultConfig() *Config {
	return &Config{
		Words: defaultTestName,
		Time: defaultTestDuration,
		GameMode: defaultGameMode,
		data: defaultDataDir,
		NumWordsPerLine: defaultNumWordsPerLine,
		WindowSize: defaultWindowSize,
		AllowBackspace: defaultAllowBackspace,
		TestSize: defaultTestSize,
		WordsTestSize: defaultWordsTestSize,
	}
}

func DefaultConfig2() *Config2 {
	return &Config2{
		TestName: defaultTestName,
		TestDuration: defaultTestDuration,
		GameMode: defaultGameMode,
		data: defaultDataDir,
		NumWordsPerLine: defaultNumWordsPerLine,
		WindowSize: defaultWindowSize,
		AllowBackspace: defaultAllowBackspace,
		TestSize: defaultTestSize,
		WordsTestSize: defaultWordsTestSize,
		CursorColor: defaultCursorColor,
		MatchColor: defaultMatchColor,
		MismatchColor: defaultMismatchColor,
		LineSpacing: defaultLineSpacing,
		DefaultColor: defaultColor,
		OverlapSpaceColor: defaultOverlapSpaceColor,
	}
}

func initializeConfigDir() (*Config2, error) {
	if err := os.Mkdir(defaultConfigDir, 0777); err != nil {
		return nil, err
	}

	config := DefaultConfig2()

	file, err := os.Create(defaultConfigPath)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	if err := config.write(file); err != nil {
		return nil, err
	}

	if err := os.Mkdir(defaultDataDir, 0777); err != nil {
		return nil, err
	}

	return config, nil
}

func ReadOrCreateConfig() (*Config2, error) {
	var dirNotFound bool
	_, err := os.Lstat(defaultConfigDir)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			dirNotFound = true
		} else {
			return nil, err
		}
	}

	if dirNotFound {
		return initializeConfigDir()
	}

	var testDirNotFound bool

	if _, err := os.Lstat(defaultTestDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			testDirNotFound = true
		} else {
			return nil, err
		}
	}

	if testDirNotFound {
		if err := os.Mkdir(defaultTestDir, 0777); err != nil {
			return nil, err
		}
	}

	return ReadConfigFile2()
}
