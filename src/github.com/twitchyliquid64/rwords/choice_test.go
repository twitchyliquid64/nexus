package rwords

import (
	"math/rand"
	"sort"
	"testing"
)

func TestChoice(t *testing.T) {
	weights := []wordWeight{
		wordWeight{"high", 0.6},
		wordWeight{"low", 0.1},
	}
	sort.Sort(ByWeight(weights))

	r := rand.New(rand.NewSource(4532))
	result := weightedChoice(weights, r)
	if result.word != "high" {
		t.Error("Expected high, got", result.word)
	}

	r = rand.New(rand.NewSource(45))
	result = weightedChoice(weights, r)
	if result.word != "low" {
		t.Error("Expected low, got", result.word)
	}
}

func TestChoiceDistribution(t *testing.T) {
	weights := []wordWeight{
		wordWeight{"high", 0.6},
		wordWeight{"low", 0.1},
		wordWeight{"med", 0.3},
	}
	sort.Sort(ByWeight(weights))

	r := rand.New(rand.NewSource(4657543))

	output := map[string]int{}
	for i := 0; i < 1000; i++ {
		result := weightedChoice(weights, r)
		output[result.word] = output[result.word] + 1
	}

	t.Logf("Observed: %+v", output)
	if (output["low"]*4) > output["high"] || output["med"] < output["low"] || output["high"] < output["med"] {
		t.Error("Bad distribution")
	}
}
