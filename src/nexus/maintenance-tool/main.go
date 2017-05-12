package main

import (
	"context"
	"flag"
	"fmt"
	"nexus/data"
	"os"
)

func die(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}

func main() {
	flag.Parse()

	if len(flag.Args()) < 1 {
		die("USAGE: %s <db-path> [command]")
	}
	fmt.Printf("Attempting open of db: %q: ", flag.Arg(0))
	db, err := data.Init(context.Background(), "ql", flag.Arg(0))
	if err != nil {
		die(fmt.Sprintf("ERROR!\nError: %s", err.Error()))
	}
	fmt.Println("OK")
	defer db.Close()

}
