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
