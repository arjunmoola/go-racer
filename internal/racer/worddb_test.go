package racer

import (
	"testing"
)

func TestLoadWordDb(t *testing.T) {
	path := "testdata/words"

	wordDb, err := LoadWordDb(path)

	if err != nil {
		t.Errorf("got error: %v", err)
		t.FailNow()
	}

	if n := len(wordDb.wordLists); n != 1 {
		t.Errorf("incorrect number of wordlists in wordDb got %d wanted %d", n, 1)
	}

	key := "english_10k"

	_, ok := wordDb.Get(key)

	if !ok {
		t.Errorf("could not find word list in db for key %s", key)
	}
}


