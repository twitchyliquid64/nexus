package dlock

import (
	"sync"
)

var l sync.RWMutex

func Lock() *sync.RWMutex {
	return &l
}
