package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
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

var (
	dbFlag          = flag.String("db", "dev.db", "path to the database file")
	nameFlag        = flag.String("name", "", "")
	usernameFlag    = flag.String("username", "", "")
	kindFlag        = flag.String("kind", "", "")
	caCertPathFlag  = flag.String("ca_cert", "", "Path to CA cert")
	caKeyPathFlag   = flag.String("ca_key", "", "Path to CA key")
	cliCertPathFlag = flag.String("client_cert", "", "Path to client cert (to mint)")
	cliKeyPathFlag  = flag.String("client_key", "", "Path to client key (to mint)")
)

var commandTable = map[string]func(context.Context, *sql.DB) error{
	"CREATEUSER":        createUserCommand,
	"RESETAUTH":         resetAuthCommand,
	"CREATESESSION":     createSession,
	"LISTSESSIONS":      listSessions,
	"ADDMSGSOURCE":      addMessagingSource,
	"LISTMSGSOURCES":    listMessagingSources,
	"LISTGRANTS":        listGrants,
	"LISTDATASTORES":    listDatastores,
	"CREATEGRANT":       createGrant,
	"CREATECA":          createCaCertCommand,
	"MINTCLIENTCERT":    mintClientCertCommand,
	"GETUSERLISTREMOTE": getUserListRemote,
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

	cmd := strings.Replace(strings.ToUpper(flag.Arg(0)), "-", "", -1)
	_, commandExists := commandTable[cmd]
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

	cmd := strings.Replace(strings.ToUpper(flag.Arg(0)), "-", "", -1)
	err = commandTable[cmd](ctx, db)
	fmt.Printf("%s: ", flag.Arg(0))
	if err != nil {
		fmt.Println("ERROR!")
		fmt.Println(err)
		os.Exit(1)
	} else {
		fmt.Println("OK.")
	}
}

func getUserListRemote(ctx context.Context, db *sql.DB) error {
	if *cliCertPathFlag == "" {
		*cliCertPathFlag = prompt("File name of client cert (to use) [client_cert.pem]")
	}
	if *cliCertPathFlag == "" {
		*cliCertPathFlag = "client_cert.pem"
	}
	if *cliKeyPathFlag == "" {
		*cliKeyPathFlag = prompt("File name of client key (to use) [client_key.pem]")
	}
	if *cliKeyPathFlag == "" {
		*cliKeyPathFlag = "client_key.pem"
	}
	var serverAddress string
	for serverAddress == "" {
		serverAddress = prompt("Server address")
	}

	// Load client cert
	cert, err := tls.LoadX509KeyPair(*cliCertPathFlag, *cliKeyPathFlag)
	if err != nil {
		log.Fatal(err)
	}

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}
	req, err := http.NewRequest("GET", "https://"+serverAddress+"/federation/v1/accounts/users", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	io.Copy(os.Stdout, resp.Body)
	fmt.Println()
	return nil
}

func createCaCertCommand(ctx context.Context, db *sql.DB) error {
	if *caCertPathFlag == "" {
		*caCertPathFlag = prompt("Path to CA Cert file [/etc/subnet/ca_cert.pem]")
	}
	if *caCertPathFlag == "" {
		*caCertPathFlag = "/etc/subnet/ca_cert.pem"
	}

	if *caKeyPathFlag == "" {
		*caKeyPathFlag = prompt("Path to CA Key file [/etc/subnet/ca_key.pem]")
	}
	if *caKeyPathFlag == "" {
		*caKeyPathFlag = "/etc/subnet/ca_key.pem"
	}

	if _, statErr := os.Stat(path.Dir(*caCertPathFlag)); path.IsAbs(*caCertPathFlag) && os.IsNotExist(statErr) {
		return fmt.Errorf("Directory %q does not exist", path.Dir(*caCertPathFlag))
	}
	if _, statErr := os.Stat(path.Dir(*caKeyPathFlag)); path.IsAbs(*caKeyPathFlag) && os.IsNotExist(statErr) {
		return fmt.Errorf("Directory %q does not exist", path.Dir(*caKeyPathFlag))
	}

	return makeCaCert(*caCertPathFlag, *caKeyPathFlag)
}

func mintClientCertCommand(ctx context.Context, db *sql.DB) error {
	if *caCertPathFlag == "" {
		*caCertPathFlag = prompt("Path to CA Cert file [/etc/subnet/ca_cert.pem]")
	}
	if *caCertPathFlag == "" {
		*caCertPathFlag = "/etc/subnet/ca_cert.pem"
	}
	if *caKeyPathFlag == "" {
		*caKeyPathFlag = prompt("Path to CA Key file [/etc/subnet/ca_key.pem]")
	}
	if *caKeyPathFlag == "" {
		*caKeyPathFlag = "/etc/subnet/ca_key.pem"
	}

	if *cliCertPathFlag == "" {
		*cliCertPathFlag = prompt("File name of client cert (to mint) [client_cert.pem]")
	}
	if *cliCertPathFlag == "" {
		*cliCertPathFlag = "client_cert.pem"
	}
	if *cliKeyPathFlag == "" {
		*cliKeyPathFlag = prompt("File name of client key (to mint) [client_key.pem]")
	}
	if *cliKeyPathFlag == "" {
		*cliKeyPathFlag = "client_key.pem"
	}

	if _, statErr := os.Stat(path.Dir(*cliCertPathFlag)); path.IsAbs(*cliCertPathFlag) && os.IsNotExist(statErr) {
		return fmt.Errorf("Directory %q does not exist", path.Dir(*cliCertPathFlag))
	}
	if _, statErr := os.Stat(path.Dir(*cliKeyPathFlag)); path.IsAbs(*cliKeyPathFlag) && os.IsNotExist(statErr) {
		return fmt.Errorf("Directory %q does not exist", path.Dir(*cliKeyPathFlag))
	}

	return issueClientCert(*caCertPathFlag, *caKeyPathFlag, *cliCertPathFlag, *cliKeyPathFlag)
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

	sid, err := session.Create(ctx, usr.UID, true, true, session.Admin, "{\"Score\": 1000}", db)
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
