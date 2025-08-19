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

type Config struct {
	Words string `json:"words"`
	Time int `json:"time"`
	Mode string `json:"mode"`
	AllowBackspace string `json:"allowBackspace"`
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

	if config.AllowBackspace == "" {
		config.AllowBackspace = "yes"
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
		Words: "english",
		Time: 30,
		data: defaultDataDir,
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

	if err := os.Mkdir(defaultTestDir, 0777); err != nil {
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
