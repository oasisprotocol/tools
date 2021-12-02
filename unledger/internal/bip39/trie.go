package bip39

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
)

const wordListLength = 2048

var (
	//go:embed english.txt
	englishWordList []byte
	englishWordLUT  map[string]int
	englishTrie     *trieNode
)

// ErrAmbiguous is the error returned when a mnemonic word is ambiguous.
var ErrAmbiguous = fmt.Errorf("bip39: ambiguous mnemonic word")

type trieNode struct {
	isLeaf   bool
	children map[byte]*trieNode
}

// Lookup looks up a prefix, and returns the full word or ErrAmbiguous
// if the prefix is insufficiently long.
func (n *trieNode) Lookup(prefix string) (string, error) {
	prefix = strings.ToLower(prefix)

	var (
		ret string
		p   *trieNode = n
	)
	for _, c := range prefix {
		if p == nil || c < 'a' || c > 'z' {
			// Either the prefix is too long, or one or more of the
			// characters is out of range.
			return "", fmt.Errorf("bip39: invalid mnemonic word")
		}
		ret = ret + string(c)
		p = p.children[byte(c)]
	}
	if p == nil || p.isLeaf {
		return ret, nil
	}
	for {
		switch len(p.children) {
		case 0:
			if !p.isLeaf {
				panic("BUG: bip39: trie corruption, no children, not leaf: '" + prefix + "'")
			}
			return ret, nil
		case 1:
			for k, v := range p.children {
				ret = ret + string(k)
				p = v
				continue
			}
		default:
			if p.isLeaf {
				panic("BUG: bip39: trie corruption, multiple children, is leaf: '" + prefix + "'")
			}
			return "", ErrAmbiguous
		}
	}
}

// Insert inserts a word into the wordlist.
func (n *trieNode) Insert(s string) {
	c := s[0]
	child := n.children[c]
	if child == nil {
		child = &trieNode{
			children: make(map[byte]*trieNode),
		}
		n.children[c] = child
	}
	if len(s) > 1 {
		child.Insert(s[1:])
	} else {
		child.isLeaf = true
	}
}

// ExpandWord expands a potentially abreviated mnemonic word to the full word.
func ExpandWord(prefix string) (string, error) {
	return englishTrie.Lookup(strings.ToLower(prefix))
}

func init() {
	englishWordLUT = make(map[string]int)

	englishTrie = &trieNode{
		children: make(map[byte]*trieNode),
	}
	words := bytes.Fields(englishWordList)
	for i, word := range words {
		englishTrie.Insert(string(word))
		englishWordLUT[string(word)] = i
	}
	if len(words) != wordListLength {
		panic("BUG: bip39: word list is not 2048-entries long")
	}
	if len(englishWordLUT) != wordListLength {
		panic("BUG: bip39: word LUT is not 2048-entries long")
	}
}
