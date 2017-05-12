package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/olekukonko/tablewriter"

	"nexus/data"
	"nexus/data/session"
	"nexus/data/user"
)

func die(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}

var dbFlag = flag.String("db", "dev.db", "path to the database file")
var nameFlag = flag.String("name", "", "")
var usernameFlag = flag.String("username", "", "")

var commandTable = map[string]func(context.Context, *sql.DB) error{
	"CREATEUSER":    createUserCommand,
	"RESETAUTH":     resetAuthCommand,
	"CREATESESSION": createSession,
	"LISTSESSIONS":  listSessions,
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
		fmt.Println("OK.")
	}
}

func createUserCommand(ctx context.Context, db *sql.DB) error {
	if *nameFlag == "" {
		die("Error: createuser needs name flag: --name <display name>")
	}
	if *usernameFlag == "" {
		die("Error: createuser needs username flag: --username <username>")
	}

	uid, displayName, createdAt, err := user.Get(ctx, *usernameFlag, db)
	if err == nil {
		fmt.Println("Error: User already exists!")
		fmt.Printf("  uid=%d, display_name=%q, created_at=%q\n", uid, displayName, createdAt)
		die("")
	} else if err != user.ErrUserDoesntExist {
		return err
	}

	return user.Create(ctx, *usernameFlag, *nameFlag, db)
}

func resetAuthCommand(ctx context.Context, db *sql.DB) error {
	if *usernameFlag == "" {
		die("Error: resetauth needs username flag: --username <username>")
	}

	if len(flag.Arg(1)) < 6 {
		return errors.New("Password too short (6 char minimum)")
	}

	uid, _, _, err := user.Get(ctx, *usernameFlag, db)
	if err != nil {
		return err
	}

	return user.SetAuth(ctx, uid, flag.Arg(1), db)
}

func createSession(ctx context.Context, db *sql.DB) error {
	if *usernameFlag == "" {
		die("Error: createSession needs username flag: --username <username>")
	}

	uid, _, _, err := user.Get(ctx, *usernameFlag, db)
	if err != nil {
		return err
	}

	sid, err := session.Create(ctx, uid, true, true, session.Admin, db)
	if err == nil {
		fmt.Printf("\nSession = %q\n", sid)
	}
	return err
}

func listSessions(ctx context.Context, db *sql.DB) error {
	if *usernameFlag == "" {
		die("Error: createSession needs username flag: --username <username>")
	}

	uid, displayName, _, err := user.Get(ctx, *usernameFlag, db)
	if err != nil {
		return err
	}

	sessions, err := session.GetAllForUser(ctx, uid, db)
	if err != nil {
		return err
	}
	fmt.Printf("\nShowing sessions for %q (uid=%d)\n", displayName, uid)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"#", "SID", "Creation time", "Web?", "API?", "Revoked", "Authentication method"})
	table.SetFooter([]string{"", "", "", "", "", "Total", strconv.Itoa(len(sessions)) + tablewriter.ConditionString(len(sessions) != 1, " sessions", " session")})
	table.SetAutoMergeCells(true)
	table.SetBorder(false)
	table.SetRowLine(true)
	for i, session := range sessions {
		var row []string
		row = append(row, strconv.Itoa(i+1))
		row = append(row, session.SID)
		row = append(row, session.Created.Format(time.Stamp))
		row = append(row, tablewriter.ConditionString(session.AccessWeb, "yes", "no"))
		row = append(row, tablewriter.ConditionString(session.AccessAPI, "yes", "no"))
		row = append(row, tablewriter.ConditionString(session.Revoked, "yes", "no"))
		row = append(row, string(session.AuthedVia))
		table.Append(row)
	}
	table.Render()
	return nil
}
