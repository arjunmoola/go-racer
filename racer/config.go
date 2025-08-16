package racer

import (
	"io"
	"os"
	"bufio"
	"strings"
	"errors"
	"strconv"
	"fmt"
)

var (
	ErrInvalidConfig = errors.New("invalid config")
	ErrInvalidConfigKey = errors.New("invalid config key")
)

const (
	defaultPath = "$HOME/.go-racer.conf"
	defaultDataDir = "$HOME/.local/share/go-racer/data"
)

type Config struct {
	words string
	time int
	mode string
	data string
}

func ReadConfigFile() (*Config, error) {
	path := "$HOME/.go-racer.conf"

	file, err := os.Open(os.ExpandEnv(path))

	if err != nil {
		return nil, err
	}

	defer file.Close()

	return parseConfig(file)
}

func parseConfig(r io.Reader) (*Config, error) {
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanLines)


	config := &Config{}

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")

		if len(parts) != 2 {
			return nil, ErrInvalidConfig
		}

		key, value := parts[0], parts[1]
		value = strings.TrimSpace(value)

		switch key {
		case "words":
			config.words = value
		case "time":
			v, _ := strconv.ParseInt(value, 10, 64)
			config.time = int(v)
		case "mode":
			config.mode = value
		case "data":
			config.data = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) write(w io.Writer) error {
	if c.words != "" {
		fmt.Fprintf(w, "words:%s\n", c.words)
	} else {
		fmt.Fprintln(w, "words:english")
	}

	if c.time != 0 {
		fmt.Fprintf(w, "time:%d\n", c.time)
	} else{
		fmt.Fprintln(w, "time:30")
	}

	if c.data != "" {
		fmt.Fprintf(w, "data:%s\n", c.data)
	} else {
		fmt.Fprintln(w, defaultDataDir)
	}

	return nil
}

func (c *Config) Save() error {
	file, err := os.Create(os.ExpandEnv(defaultPath))

	if err != nil {
		return err
	}

	defer file.Close()

	return c.write(file)
}

func ReadOrCreateConfig() (*Config, error) {
	_, err := os.Lstat(os.ExpandEnv(defaultPath))
	fileNotFound := false

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fileNotFound = true
		} else {
			return nil, err
		}
	}

	if fileNotFound {
		config := &Config{
			words: "english",
			time: 30,
			data: defaultDataDir,
		}

		file, err := os.Create(os.ExpandEnv(defaultPath))
		
		if err != nil {
			return nil, err
		}

		defer file.Close()

		if err := config.write(file); err != nil {
			return nil, err
		}

		return config, nil
	}

	return ReadConfigFile()
}
