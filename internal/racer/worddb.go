package racer

import (
	"encoding/json"
	"os"
	"io"
	"path/filepath"
	"golang.org/x/sync/errgroup"
	"errors"

)

var (
	ErrWordDirNotFound = errors.New("word dir not setup")
)


type WordList struct {
	Name string `json:"name"`	
	NoLazyMode bool `json:"noLazyMode"`
	OrderedByFrequency bool `json:"orderedByFrequency"`
	Words []string `json:"words"`
}

type WordDb struct {
	wordLists map[string]*WordList
}

func readWordList(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

func LoadWordDb(dirPath string) (*WordDb, error) {
	dirEntries, err := os.ReadDir(dirPath)

	if err != nil {
		return nil, err
	}

	var g errgroup.Group

	output := make(chan *WordList)
	errCh := make(chan error, 1)

	for _, entry := range dirEntries {
		g.Go(func() error {
			file, err := os.Open(filepath.Join(dirPath, entry.Name()))

			if err != nil {
				return err
			}

			defer file.Close()

			wordList := &WordList{}

			if err := readWordList(file, wordList); err != nil {
				return err
			}

			output <- wordList

			return nil

		})
	}

	go func() {
		errCh <- g.Wait()
		close(output)
	}()

	db := make(map[string]*WordList)

	for wl := range output {
		db[wl.Name] = wl
	}

	if err := <-errCh; err != nil {
		return nil, err
	}

	wordDb := &WordDb{
		wordLists: db,
	}

	return wordDb, nil
}

func (w *WordDb) Get(name string) (*WordList, bool) {
	l, ok := w.wordLists[name]
	return l, ok
}

func (w *WordDb) GetWords(name string) ([]string, bool) {
	l, ok := w.wordLists[name]

	if !ok {
		return nil, false
	}

	return l.Words, true
}

func (w *WordDb) Set(l *WordList) {
	w.wordLists[l.Name] = l
}

func (w *WordDb) Contains(name string) bool {
	_, ok := w.wordLists[name]
	return ok
}

func (w *WordList) Save() error {
	path := filepath.Join(defaultDataDir, w.Name + ".json")

	file, err := os.Create(path)

	if err != nil {
		return err
	}

	defer file.Close()

	return json.NewEncoder(file).Encode(w)
}
