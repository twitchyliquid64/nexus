package terminal

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"nexus/data/datastore"
	"strings"
	"time"

	"github.com/xwb1989/sqlparser"
)

type sqlResult struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`

	Affected int64 `sql:"affected"`
}

// RunSQL executes an SQL query in the context of a user and the
// datastores they can access.
func RunSQL(ctx context.Context, db *sql.DB, query string, uid int) (*sqlResult, error) {
	stmt, err := sqlparser.Parse(query)
	if err != nil {
		return nil, err
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

		case *sqlparser.Insert:
			e := n.Table
			if len(e.Name.String()) > 0 {
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

				n.Table = sqlparser.TableName{Name: sqlparser.NewTableIdent(fmt.Sprintf("ds_%d", ds.UID)), Qualifier: e.Qualifier}

				for i := range n.Columns {
					if n.Columns[i].String() != "rowid" {
						n.Columns[i] = sqlparser.NewColIdent(columnName(n.Columns[i].String()))
					}
				}
			}

		case *sqlparser.ColName:
			if len(n.Name.String()) > 0 && n.Name.String() != "rowid" {
				n.Name = sqlparser.NewColIdent(columnName(n.Name.String()))
			}
		}

		return true, nil
	}, stmt); err != nil {
		return nil, err
	}

	sanitizedQuery := sqlparser.String(stmt)
	log.Printf("New query: %s", sanitizedQuery)

	switch sqlparser.Preview(sanitizedQuery) {
	case sqlparser.StmtSelect:
		return dynamicQuery(ctx, db, sanitizedQuery)
	case sqlparser.StmtUpdate:
		fallthrough
	case sqlparser.StmtInsert:
		res, err := db.ExecContext(ctx, sanitizedQuery)
		if err != nil {
			return nil, err
		}
		affected, _ := res.RowsAffected()
		return &sqlResult{
			Affected: affected,
		}, nil
	default:
		return nil, fmt.Errorf("query type not supported: %d", sqlparser.Preview(sanitizedQuery))
	}
}

func dynamicQuery(ctx context.Context, db *sql.DB, sanitizedQuery string) (*sqlResult, error) {
	rows, err := db.QueryContext(ctx, sanitizedQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	count := len(cols)
	values := make([]interface{}, count)
	valuePtrs := make([]interface{}, count)
	var out [][]interface{}
	for rows.Next() {
		var row []interface{}
		for i := range cols {
			valuePtrs[i] = &values[i]
		}

		if err = rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		for i, _ := range cols {
			var value interface{}
			switch v := values[i].(type) {
			case []byte:
				value = string(v)
			case time.Time:
				value = v.Format(time.RFC3339)
			default:
				value = fmt.Sprintf("%v", v)
			}
			row = append(row, value)
		}
		out = append(out, row)
	}

	return &sqlResult{
		Columns: cols,
		Rows:    out,
	}, rows.Err()
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
