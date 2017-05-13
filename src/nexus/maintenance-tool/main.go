package main

import (
	"context"
	"database/sql"
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
	db, err := data.Init(ctx, "ql", *dbFlag)
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

	usr, err := user.Get(ctx, *usernameFlag, db)
	if err == nil {
		fmt.Println("Error: User already exists!")
		fmt.Printf("  uid=%d, display_name=%q, created_at=%q\n", usr.UID, usr.DisplayName, usr.CreatedAt)
		die("")
	} else if err != user.ErrUserDoesntExist {
		return err
	}

	return user.CreateBasic(ctx, *usernameFlag, *nameFlag, db)
}

func booleanPrompt(question string) bool {
	for {
		r := prompt(question + " [y/N]")
		switch r {
		case "N":
			return false
		case "":
			return false
		case "y":
			return true
		}
	}
}

func prompt(question string) string {
	fmt.Print(question + ": ")
	var input string
	fmt.Scanln(&input)
	return input
}

func resetAuthCommand(ctx context.Context, db *sql.DB) error {
	if *usernameFlag == "" {
		die("Error: resetauth needs username flag: --username <username>")
	}

	usr, err := user.Get(ctx, *usernameFlag, db)
	if err != nil {
		return err
	}

	pw := prompt("Users password")
	accAdmin := booleanPrompt("Allowed to manage accounts?")
	dataAdmin := booleanPrompt("Allowed to manage data?")
	integrationAdmin := booleanPrompt("Allowed to manage integrations?")

	return user.SetAuth(ctx, usr.UID, pw, accAdmin, dataAdmin, integrationAdmin, db)
}

func createSession(ctx context.Context, db *sql.DB) error {
	if *usernameFlag == "" {
		die("Error: createSession needs username flag: --username <username>")
	}

	usr, err := user.Get(ctx, *usernameFlag, db)
	if err != nil {
		return err
	}

	sid, err := session.Create(ctx, usr.UID, true, true, session.Admin, db)
	if err == nil {
		fmt.Printf("\nSession = %q\n", sid)
	}
	return err
}

func listSessions(ctx context.Context, db *sql.DB) error {
	if *usernameFlag == "" {
		die("Error: createSession needs username flag: --username <username>")
	}

	usr, err := user.Get(ctx, *usernameFlag, db)
	if err != nil {
		return err
	}

	sessions, err := session.GetAllForUser(ctx, usr.UID, db)
	if err != nil {
		return err
	}
	fmt.Printf("\nShowing sessions for %q (uid=%d)\n", usr.DisplayName, usr.UID)

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
