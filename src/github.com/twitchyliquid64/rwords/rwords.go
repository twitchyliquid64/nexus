package rwords

import (
	"math/rand"
	"sort"
	"strings"
	"time"
)

var (
	vowels = []string{"a", "e", "i", "o", "u"}
	consts = []string{"b", "c", "d", "f", "g", "h", "j", "k", "l", "m", "n", "p", "qu", "r", "s", "t", "v", "w", "x", "y", "z", "tt", "ch", "sh"}
)

func buildMap(input string) map[byte][]wordWeight {
	input = strings.NewReplacer(".", " ", ",", " ", "\n", " ", "\t", " ", "'", "", "\"", "").Replace(strings.ToLower(input))
	spl := strings.Split(input, " ")
	incidenceMap := map[byte][]wordWeight{}
	for _, word := range spl {
		if word == "" || word == "\n" {
			continue
		}
		word = strings.TrimSpace(word)

		for i := 0; i < len(word); i++ {
			c := word[i]
			var next string
			if (i + 1) < len(word) {
				next = string(word[i+1])
			}

			found := false
			for i := range incidenceMap[c] {
				if incidenceMap[c][i].word == string(next) {
					incidenceMap[c][i].weight++
					found = true
					break
				}
			}
			if !found {
				incidenceMap[c] = append(incidenceMap[c], wordWeight{next, 1})
			}
		}
	}
	return incidenceMap
}

func sortMap(in map[byte][]wordWeight) map[byte][]wordWeight {
	out := map[byte][]wordWeight{}
	for k, v := range in {
		s := v
		sort.Sort(ByWeight(s))
		out[k] = s
	}
	return out
}

// RandomMarkov generates a word based on the markov distribution of letters in the space-delimited training set.
func RandomMarkov(training string) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	m := sortMap(buildMap(training))
	var cursor byte
	var out string

	for k, _ := range m {
		cursor = k
		break
	}

	out += string(cursor)
	for {
		choice := weightedChoice(m[cursor], r)
		if len(choice.word) == 0 {
			break
		}
		cursor = byte(choice.word[0])
		out += choice.word
	}
	return out
}

// RandomSimple returns a random pronouncable word with the given number of sound components.
func RandomSimple(sounds int) string {
	return randomSimple(sounds, rand.New(rand.NewSource(time.Now().UnixNano())))
}

func randomSimple(sounds int, rand *rand.Rand) string {
	isVowel := false
	out := ""

	for i := 0; i < sounds; i++ {
		var source []string
		if isVowel {
			source = vowels
		} else {
			source = consts
		}

		isVowel = !isVowel

		out += source[rand.Intn(len(source)-1)]
	}
	return out
}
