package bip39

import (
	"bytes"
	"testing"
)

func TestTrie(t *testing.T) {
	// Exhaustively lookup every single possible prefix.
	for _, wordBytes := range bytes.Fields(englishWordList) {
		word := string(wordBytes)
		prefix, suffix := word, ""
		if len(prefix) > 4 {
			prefix = word[:4]
			suffix = word[4:]
		}
		fullWord, err := englishTrie.Lookup(prefix)
		if err != nil {
			t.Errorf("Lookup(%s): %v", prefix, err)
			continue
		}
		if word != fullWord {
			t.Errorf("Lookup(%s) != '%s', got %s", prefix, word, fullWord)
		}

		if len(prefix) != len(word) {
			longPrefix := prefix
			for _, c := range suffix {
				longPrefix += string(c)
				fullWord, err = englishTrie.Lookup(longPrefix)
				if err != nil {
					t.Errorf("Lookup(%s): %v", longPrefix, err)
					break
				}
				if word != fullWord {
					t.Errorf("Lookup(%s) != '%s', got %s", longPrefix, word, fullWord)
					break
				}
			}
		}
	}
}
