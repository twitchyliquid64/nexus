package terminal

import (
	"context"
	"database/sql"
  "fmt"
  "log"
  "nexus/data/datastore"
  "strings"

	"github.com/xwb1989/sqlparser"
)

// RunSQL executes an SQL query in the context of a user and the
// datastores they can access.
func RunSQL(ctx context.Context, db *sql.DB, query string, uid int) error {
	stmt, err := sqlparser.Parse(query)
	if err != nil {
		return err
	}

  dUID := 0
  // Rewrite Table/Column names, check ownership of tables.
	if err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		log.Printf("Node %T: %s", node, node)

    switch n := node.(type) {
    case *sqlparser.AliasedTableExpr:
      e, ok := n.Expr.(sqlparser.TableName)
      if ok && len(e.Name.String()) > 0 {
        ds, err := datastore.GetDatastoreByName(ctx, e.Name.String(), db)
        if err != nil {
          return false, err
        }
        if ds.OwnerID != uid {
          return false, fmt.Errorf("referenced datastore %q is not owned by you", ds.Name)
        }

        if dUID != 0 && dUID != ds.UID {
          return false, fmt.Errorf("only one datastore may be referenced in a query")
        }
        dUID = ds.UID

        n.Expr = sqlparser.TableName{Name: sqlparser.NewTableIdent(fmt.Sprintf("ds_%d", ds.UID)), Qualifier: e.Qualifier}
      }

    case *sqlparser.ColName:
      if len(n.Name.String()) > 0 {
        n.Name = sqlparser.NewColIdent(columnName(n.Name.String()))
      }
    }

		return true, nil
	}, stmt); err != nil {
    return err
  }

  log.Printf("New query: %s", sqlparser.String(stmt))

	return nil
}

func columnName(inName string) string {
	c := strings.Replace(inName, " ", "_", -1) + "_"
	o := make([]rune, len(c))
	for i, char := range c {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			o[i] = char
		} else {
			o[i] = '_'
		}
	}
	return "C" + string(o)
}
