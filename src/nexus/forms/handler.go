package forms

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
)

// HandleSubmission handles form submissions.
func HandleSubmission(req *http.Request, formID string, userID int, db *sql.DB) error {
	for _, source := range getForms(false) {
		for _, action := range source.Actions() {
			if f, isForm := action.(form); isForm && f.UniqueID() == formID {
				log.Printf("[forms] User %d submitted %q", userID, f.Title())

				err := req.ParseForm()
				if err != nil {
					return err
				}

				fields := map[string]string{}
				for k, v := range req.Form {
					fields[k] = v[0]
				}

				if f.OnSubmitHandler() == nil {
					return fmt.Errorf("Nil handler for form %q (%s)", f.Title(), formID)
				}
				return f.OnSubmitHandler()(req.Context(), fields, userID, db)
			}
		}
	}
	return nil
}

// HandleTableAction handles button presses in the tables.
func HandleTableAction(rowID, formID, actionUID string, userID int, db *sql.DB) (string, error) {
	for _, source := range getForms(false) {
		for _, cs := range source.GetContentSections() {
			if f, isTable := cs.(table); isTable && f.UniqueID() == formID {
				log.Printf("[forms] User %d pressed button on %q", userID, f.Title())

				for _, tableAction := range f.GetActions() {
					if a, isAction := tableAction.(action); isAction && a.UniqueID() == actionUID {
						if a.OnSubmitHandler() == nil {
							return "", fmt.Errorf("Nil handler for table handler %q (%s)", f.Title(), formID)
						}
						return source.UniqueID(), a.OnSubmitHandler()(rowID, formID, actionUID, userID, db)
					}
				}
			}
		}
	}
	return "", nil
}
