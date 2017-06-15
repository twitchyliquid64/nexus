package integration

import (
	"fmt"

	python "github.com/sbinet/go-python"
)

func init() {
	err := python.Initialize()
	if err != nil {
		panic(err.Error())
	}
}

// Test succeeds if Python works / initializes successfully.
func Test() {
	gostr := "from python"
	pystr := python.PyString_FromString(gostr)
	str := python.PyString_AsString(pystr)
	fmt.Println("hello [", str, "]")
}
