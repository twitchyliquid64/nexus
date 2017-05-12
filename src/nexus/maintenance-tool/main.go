package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"nexus/data"
	"nexus/data/user"
	"os"
	"strings"
	"text/tabwriter"
)

func die(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}

var dbFlag = flag.String("db", "dev.db", "path to the database file")
var nameFlag = flag.String("name", "", "")
var usernameFlag = flag.String("username", "", "")

var commandTable = map[string]func(context.Context, *sql.DB) error{
	"CREATEUSER": createUserCommand,
	"RESETAUTH":  resetAuthCommand,
}

func printCommands() {
	fmt.Println("Commands:")

	w := tabwriter.NewWriter(os.Stdout, 8, 3, 1, '\t', tabwriter.TabIndent|tabwriter.StripEscape)
	counter := 0
	for command := range commandTable {
		counter++
		fmt.Fprint(w, command+"\t")
		if counter%4 == 0 {
			fmt.Fprintln(w)
		}
	}

	if counter%4 != 0 {
		fmt.Fprintln(w)
	}
	w.Flush()
	os.Exit(1)
}

func validateCommandLine() {
	flag.Parse()

	if len(flag.Args()) < 1 {
		die(fmt.Sprintf("USAGE: %s [--db <db-path>] command <command-specific-args>", os.Args[0]))
	}

	_, commandExists := commandTable[strings.ToUpper(flag.Arg(0))]
	if !commandExists {
		fmt.Printf("Error: %q is not a valid command.\n", flag.Arg(0))
		printCommands()
	}
}

func main() {
	validateCommandLine()
	ctx := context.Background()

	fmt.Printf("Attempting open of db: %q: ", *dbFlag)
	db, err := data.Init(context.Background(), "ql", *dbFlag)
	if err != nil {
		die(fmt.Sprintf("ERROR!\nError: %s", err.Error()))
	}
	fmt.Println("OK")
	defer db.Close()

	fmt.Printf("%s: ", flag.Arg(0))
	err = commandTable[strings.ToUpper(flag.Arg(0))](ctx, db)
	if err != nil {
		fmt.Println("ERROR!")
		fmt.Println(err)
		os.Exit(1)
	} else {
		fmt.Println("SUCCESS.")
	}
}

func createUserCommand(ctx context.Context, db *sql.DB) error {
	if *nameFlag == "" {
		die("Error: createuser needs name flag: --name <display name>")
	}
	if *usernameFlag == "" {
		die("Error: createuser needs username flag: --username <username>")
	}

	uid, displayName, createdAt, err := user.GetUser(ctx, *usernameFlag, db)
	if err == nil {
		fmt.Println("Error: User already exists!")
		fmt.Printf("  uid=%d, display_name=%q, created_at=%q\n", uid, displayName, createdAt)
		die("")
	} else if err != user.ErrUserDoesntExist {
		return err
	}

	return user.CreateUser(ctx, *usernameFlag, *nameFlag, db)
}

func resetAuthCommand(ctx context.Context, db *sql.DB) error {
	if *usernameFlag == "" {
		die("Error: resetauth needs username flag: --username <username>")
	}

	if len(flag.Arg(1)) < 6 {
		return errors.New("Password too short (6 char minimum)")
	}

	uid, _, _, err := user.GetUser(ctx, *usernameFlag, db)
	if err != nil {
		return err
	}

	return user.SetAuth(ctx, uid, flag.Arg(1), db)
}
