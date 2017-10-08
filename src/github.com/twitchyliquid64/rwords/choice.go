package rwords

import (
	"math/rand"
)

type wordWeight struct {
	word   string
	weight float64
}

// ByWeight implements sort.Interface for []wordWeight based on
// the weight field.
type ByWeight []wordWeight

func (a ByWeight) Len() int           { return len(a) }
func (a ByWeight) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByWeight) Less(i, j int) bool { return a[i].weight < a[j].weight }

// assumes weights are ordered
func weightedChoice(weights []wordWeight, source *rand.Rand) *wordWeight {
	var totalWeight float64
	for _, w := range weights {
		totalWeight += w.weight
	}
	randomValue := source.Float64() * totalWeight
	for _, w := range weights {
		randomValue -= w.weight
		if randomValue <= 0 {
			return &w
		}
	}

	return nil
}
