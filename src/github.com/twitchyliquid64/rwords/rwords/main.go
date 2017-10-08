// Example using the markov generator, based on a few strings and an except from the wiki page on Cthulhu.

package main

import (
	"fmt"

	"github.com/twitchyliquid64/rwords"
)

const training = `
the quick brown fox jumped over the lazy dog
returns an exponentially distributed float64 in the range
default Source is safe for concurrent use by multiple goroutines

With the revelation of writing detailing his relations we have learned that Cthulhu descends from Yog Sothoth possibly having been born on Vhoorl in the 23rd Nebula. He mated with yaa on the planet Xoth. His offspring are Ghatanothoa, Ythogtha, Zoth Ommog, and Cthylla.

`

func main() {
	fmt.Println(rwords.RandomMarkov(training))
}
