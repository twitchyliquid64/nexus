package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/olekukonko/tablewriter"

	"nexus/data"
	"nexus/data/datastore"
	"nexus/data/messaging"
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
var kindFlag = flag.String("kind", "", "")

var commandTable = map[string]func(context.Context, *sql.DB) error{
	"CREATEUSER":     createUserCommand,
	"RESETAUTH":      resetAuthCommand,
	"CREATESESSION":  createSession,
	"LISTSESSIONS":   listSessions,
	"ADDMSGSOURCE":   addMessagingSource,
	"LISTMSGSOURCES": listMessagingSources,
	"LISTGRANTS":     listGrants,
	"LISTDATASTORES": listDatastores,
	"CREATEGRANT":    createGrant,
}

func printCommands() {
	fmt.Println("Commands:")

	w := tabwriter.NewWriter(os.Stdout, 12, 3, 3, ' ', tabwriter.TabIndent|tabwriter.StripEscape)
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
	db, err := data.Init(ctx, *dbFlag)
	if err != nil {
		die(fmt.Sprintf("ERROR!\nError: %s", err.Error()))
	}
	fmt.Println("OK")
	defer db.Close()

	err = commandTable[strings.ToUpper(flag.Arg(0))](ctx, db)
	fmt.Printf("%s: ", flag.Arg(0))
	if err != nil {
		fmt.Println("ERROR!")
		fmt.Println(err)
		os.Exit(1)
	} else {
		fmt.Println("OK.")
	}
}

func createUserCommand(ctx context.Context, db *sql.DB) error {
	for *nameFlag == "" {
		*nameFlag = prompt("name")
	}
	for *usernameFlag == "" {
		*usernameFlag = prompt("username")
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
	for *usernameFlag == "" {
		*usernameFlag = prompt("username")
	}

	usr, err := user.Get(ctx, *usernameFlag, db)
	if err != nil {
		return err
	}

	pw := prompt("Users password")
	accAdmin := booleanPrompt("Allowed to manage accounts?")
	dataAdmin := booleanPrompt("Allowed to manage data?")
	integrationAdmin := booleanPrompt("Allowed to manage integrations?")

	err = user.SetAuth(ctx, usr.UID, pw, accAdmin, dataAdmin, integrationAdmin, db)
	if err != nil {
		return err
	}

	auths, err := user.GetAuthForUser(ctx, usr.UID, db)
	if err != nil {
		return err
	}
	for _, auth := range auths {
		err = user.DeleteAuth(ctx, auth.UID, db)
		if err != nil {
			return err
		}
	}

	return nil
}

func createSession(ctx context.Context, db *sql.DB) error {
	for *usernameFlag == "" {
		*usernameFlag = prompt("username")
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
	for *usernameFlag == "" {
		*usernameFlag = prompt("username")
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

func listDatastores(ctx context.Context, db *sql.DB) error {
	var userUID int
	if *usernameFlag != "" {
		usr, err := user.Get(ctx, *usernameFlag, db)
		if err != nil {
			return err
		}
		fmt.Printf("\nShowing datastores for %q (uid=%d)\n", usr.DisplayName, usr.UID)
		userUID = usr.UID
	}

	stores, err := datastore.GetDatastores(ctx, *usernameFlag == "", userUID, db)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"#", "UID", "Name", "Kind", "Owner", "Creation time"})
	table.SetFooter([]string{"", "", "", "", "Total", strconv.Itoa(len(stores)) + " stores"})
	table.SetBorder(false)
	table.SetRowLine(true)
	for i, ds := range stores {
		u, err := user.GetByUID(ctx, ds.OwnerID, db)
		if err != nil {
			return err
		}

		var row []string
		row = append(row, strconv.Itoa(i+1))
		row = append(row, strconv.Itoa(ds.UID))
		row = append(row, ds.Name)
		row = append(row, ds.Kind)
		row = append(row, u.Username+" (UID="+strconv.Itoa(u.UID)+")")
		row = append(row, ds.CreatedAt.Format(time.Stamp))
		table.Append(row)
	}
	table.Render()

	return nil
}

func createGrant(ctx context.Context, db *sql.DB) error {
	for *usernameFlag == "" {
		*usernameFlag = prompt("username")
	}

	var dsID string
	_, err := strconv.Atoi(dsID)
	for dsID == "" || err != nil {
		dsID = prompt("datastore UID")
		_, err = strconv.Atoi(dsID)
	}
	ds, _ := strconv.Atoi(dsID)

	readOnly := booleanPrompt("Read only")

	usr, err := user.Get(ctx, *usernameFlag, db)
	if err != nil {
		return err
	}
	fmt.Printf("Creating grant with ds_uid=%s, user_uid=%d and read_only=%v\n", dsID, usr.UID, readOnly)

	id, err := datastore.MakeGrant(ctx, &datastore.Grant{
		UsrUID:   usr.UID,
		DsUID:    ds,
		ReadOnly: readOnly,
	}, db)
	fmt.Printf("Grant ID=%d\n", id)
	return err
}

func listGrants(ctx context.Context, db *sql.DB) error {
	for *usernameFlag == "" {
		*usernameFlag = prompt("username")
	}

	usr, err := user.Get(ctx, *usernameFlag, db)
	if err != nil {
		return err
	}

	fmt.Printf("\nShowing grants for %q (uid=%d)\n", usr.DisplayName, usr.UID)

	grants, err := datastore.ListByUser(ctx, usr.UID, db)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"#", "Datastore UID", "Datastore", "Read Only", "Creation time"})
	table.SetFooter([]string{"", "", "", "Total", strconv.Itoa(len(grants)) + tablewriter.ConditionString(len(grants) != 1, " grants", " grant")})
	table.SetBorder(false)
	table.SetRowLine(true)
	for _, grant := range grants {
		ds, err := datastore.GetDatastore(ctx, grant.DsUID, db)
		if err != nil {
			return err
		}

		var row []string
		row = append(row, strconv.Itoa(grant.UID))
		row = append(row, strconv.Itoa(grant.DsUID))
		row = append(row, ds.Name)
		row = append(row, tablewriter.ConditionString(grant.ReadOnly, "yes", "no"))
		row = append(row, grant.CreatedAt.Format(time.Stamp))
		table.Append(row)
	}
	table.Render()

	return nil
}

func listMessagingSources(ctx context.Context, db *sql.DB) error {
	for *usernameFlag == "" {
		*usernameFlag = prompt("username")
	}

	usr, err := user.Get(ctx, *usernameFlag, db)
	if err != nil {
		return err
	}

	sources, err := messaging.GetAllSourcesForUser(ctx, usr.UID, db)
	if err != nil {
		return err
	}
	fmt.Printf("\nShowing messaging sources for %q (uid=%d)\n", usr.DisplayName, usr.UID)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"#", "UID", "Name", "Kind", "Remote", "Creation time", "Details"})
	table.SetFooter([]string{"", "", "", "", "", "Total", strconv.Itoa(len(sources)) + tablewriter.ConditionString(len(sources) != 1, " sources", " source")})
	table.SetBorder(false)
	table.SetRowLine(true)
	for i, source := range sources {
		var row []string
		row = append(row, strconv.Itoa(i+1))
		row = append(row, strconv.Itoa(source.UID))
		row = append(row, source.Name)
		row = append(row, source.Kind)
		row = append(row, tablewriter.ConditionString(source.Remote, "yes", "no"))
		row = append(row, source.CreatedAt.Format(time.Stamp))
		var deets []byte
		if source.Details != nil && len(source.Details) > 0 {
			deets, err = json.Marshal(source.Details)
			if err != nil {
				return err
			}
		}
		row = append(row, string(deets))
		table.Append(row)
	}
	table.Render()
	return nil
}

func addMessagingSource(ctx context.Context, db *sql.DB) error {
	for *usernameFlag == "" {
		*usernameFlag = prompt("username")
	}
	for *nameFlag == "" {
		*nameFlag = prompt("name")
	}
	for *kindFlag == "" || ((!strings.Contains(*kindFlag, messaging.Slack)) && (!strings.Contains(*kindFlag, messaging.FbMessenger)) && (!strings.Contains(*kindFlag, messaging.IRC))) {
		*kindFlag = prompt("messaging kind (slack,fb_messenger, irc)")
	}
	isRemote := booleanPrompt("In/out remotely sourced?")
	details := prompt("Additional details (JSON stanza)")

	usr, err := user.Get(ctx, *usernameFlag, db)
	if err != nil {
		return err
	}
	fmt.Println(details)

	source := messaging.Source{
		Name:    *nameFlag,
		Kind:    *kindFlag,
		OwnerID: usr.UID,
		Remote:  isRemote,
	}

	if details != "" {
		err := json.Unmarshal([]byte(details), &source.Details)
		if err != nil {
			return err
		}
	}

	return messaging.AddSource(ctx, source, db)
}
