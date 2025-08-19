package racer

import (
	"io"
	"os"
	"errors"
	"encoding/json"
	"path/filepath"
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

func initializeConfigDir() (*Config, error) {
	if err := os.Mkdir(defaultConfigDir, 0777); err != nil {
		return nil, err
	}

	config := DefaultConfig()

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

func ReadOrCreateConfig() (*Config, error) {
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

	return ReadConfigFile()
}
