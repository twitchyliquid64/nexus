package rwords

import (
	"math/rand"
	"testing"
)

var testStr = "the quick brown fox jumped over the lazy dog ExpFloat64 returns an exponentially distributed float64 in the range. default Source is safe for concurrent use by multiple goroutines"

func TestBuildMap(t *testing.T) {
	m := buildMap(testStr)
	t.Logf("Map: %+v", m)

	// one n should follow w
	if len(m['w']) != 1 || m['w'][0].word != "n" {
		t.Errorf("Expected one option for w (%d), instead got %+v", 'w', m['w'])
	}

	// four options should follow p
	if len(m['p']) != 4 {
		t.Errorf("Expected one option for p (%d), instead got %+v", 'p', m['p'])
	}

	// no punctuation
	if len(m['.']) != 0 {
		t.Errorf("Expected no options for . (%d), instead got %+v", '.', m['.'])
	}
}

func TestSortMap(t *testing.T) {
	m := sortMap(buildMap(testStr))
	t.Logf("Map: %+v", m)

	// one n should follow w
	if len(m['w']) != 1 || m['w'][0].word != "n" {
		t.Errorf("Expected one option for w (%d), instead got %+v", 'w', m['w'])
	}

	// four options should follow p
	if len(m['p']) != 4 {
		t.Errorf("Expected one option for p (%d), instead got %+v", 'p', m['p'])
	}

	// no punctuation
	if len(m['.']) != 0 {
		t.Errorf("Expected no options for . (%d), instead got %+v", '.', m['.'])
	}

	// check order
	if len(m['e']) != 8 || m['e'][7].word != "" {
		t.Errorf("Expected 8 options for e (%d) ending in '', instead got %+v", 'e', m['e'])
	}
}

func TestRandomSimple(t *testing.T) {
	type simpleWordsCase struct {
		Seed       int64
		Expected   string
		Components int
	}

	cases := []simpleWordsCase{
		simpleWordsCase{
			Seed:       4657543,
			Expected:   "wavez",
			Components: 5,
		},
		simpleWordsCase{
			Seed:       1,
			Expected:   "kowor",
			Components: 5,
		},
		simpleWordsCase{
			Seed:       9,
			Expected:   "naj",
			Components: 3,
		},
		simpleWordsCase{
			Seed:       9432345,
			Expected:   "bilechi",
			Components: 6,
		},
	}
	for i, c := range cases {
		r := rand.New(rand.NewSource(c.Seed))
		w := randomSimple(c.Components, r)
		if w != c.Expected {
			t.Errorf("[%d] Got %q, expected %q", i, w, c.Expected)
		}
	}
}
