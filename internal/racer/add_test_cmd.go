package racer

import (
	"errors"
	"flag"
	"os"
	"fmt"
	"encoding/json"
	"path/filepath"
	"golang.org/x/sync/errgroup"
)

func RunAddTest(args []string) error {
	var inputFile string
	var inputDirectory string

	cmd := flag.NewFlagSet("add-test", flag.ExitOnError)
	cmd.StringVar(&inputFile, "f", "", "input test file to add to racer program")
	cmd.StringVar(&inputDirectory, "d", "", "path of directory that contains test files to add to racer program")

	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if err := cmd.Parse(args); err != nil {
		return err
	}

	fmt.Println("running add test command")

	if inputFile == "" && inputDirectory == "" {
		fmt.Println("must provide atleast one of input file or input directory")
		flag.Usage()
		os.Exit(1)
	}

	if err := createDataDirIfNotExist(); err != nil {
		return err
	}

	if inputFile != "" {
		return processInputFile(inputFile)
	}

	if inputDirectory != "" {
		return processInputDirectory(inputDirectory)
	}

	return nil

}

func createDataDirIfNotExist() error {
	dataDir := os.ExpandEnv(defaultDataDir)
	_, err := os.Lstat(dataDir)

	dirNotFound := false

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			dirNotFound = true
		} else {
			return nil
		}
	}

	if !dirNotFound {
		return nil
	}

	return os.MkdirAll(dataDir, 0777)

}

func readInputFile(inputFile string) (*WordList, error) {
	file, err := os.Open(inputFile)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	wordList := WordList{}

	if err := json.NewDecoder(file).Decode(&wordList); err != nil {
		return nil, err
	}

	return &wordList, nil
}

func verifyWordList(wordList *WordList) error {
	if wordList.Name == "" {
		return fmt.Errorf("wordlist does not have a name")
	}

	if len(wordList.Words) == 0 {
		return fmt.Errorf("wordlist words array has no words")
	}

	return nil
}

func saveWordList(wordList *WordList) error {
	path := filepath.Join(os.ExpandEnv(defaultDataDir), wordList.Name + ".json")

	file, err := os.Create(path)

	if err != nil {
		return err
	}

	defer file.Close()

	if err := json.NewEncoder(file).Encode(wordList); err != nil {
		return err
	}

	return nil
}

func processInputFile(inputFile string) error {
	wordList, err := readInputFile(inputFile)

	if err != nil {
		return err
	}

	if err := verifyWordList(wordList); err != nil {
		return err
	}

	if err := saveWordList(wordList); err != nil {
		return err
	}

	return nil
}

func processInputDirectory(dir string) error {
	dirEntries, err := os.ReadDir(dir)

	if err != nil {
		return err
	}

	var g errgroup.Group

	for _, entry := range dirEntries {
		g.Go(func() error {
			return processInputFile(filepath.Join(dir, entry.Name()))
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}
