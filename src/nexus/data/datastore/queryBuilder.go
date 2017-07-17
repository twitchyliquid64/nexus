package datastore

import (
	"errors"
	"log"
	"strconv"
	"strings"
)

// Query encapsulates all the information needed to perform a query on a datastore.
type Query struct {
	UID     int
	Filters []Filter
	Limit   int
	Offset  int
}

// Filter represents a constraint in a query.
type Filter struct {
	Type        string
	Val         string
	Col         string
	Conditional string
}

func getColNameByUID(cols []*Column, uid string) *Column {
	uidInt, _ := strconv.Atoi(uid)
	for _, col := range cols {
		if col.UID == uidInt {
			return col
		}
	}
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

func conditional(in string) string {
	switch in {
	case ">":
		fallthrough
	case ">=":
		fallthrough
	case "<=":
		fallthrough
	case "<":
		fallthrough
	case "==":
		fallthrough
	case "!=":
		return in
	}
	return "BAD_CONDITIONAL!!!"
}

// TODO: Handle TIME type
func buildWhereQuery(cols []*Column, query Query) (string, []interface{}, error) {
	queryString := ""
	var queryParameters []interface{}

	for i, filter := range query.Filters {
		switch filter.Type {
		case "literalConstraint":
			col := getColNameByUID(cols, filter.Col)
			if col == nil {
				return "", nil, errors.New("Invalid columnID in query")
			}
			queryString += columnName(col.Name) + " " + conditional(filter.Conditional) + " ?"
			if col.Datatype == INT || col.Datatype == UINT {
				v, _ := strconv.Atoi(filter.Val)
				queryParameters = append(queryParameters, v)
			} else if col.Datatype == STR {
				queryParameters = append(queryParameters, filter.Val)
			} else if col.Datatype == BLOB {
				queryParameters = append(queryParameters, []byte(filter.Val))
			} else if col.Datatype == FLOAT {
				v, _ := strconv.ParseFloat(filter.Val, 64)
				queryParameters = append(queryParameters, v)
			} else {
				queryParameters = append(queryParameters, filter.Val)
			}

			if i < len(query.Filters)-1 {
				queryString += " AND "
			}
		}
	}

	return queryString, queryParameters, nil
}

func buildSelectQuery(cols []*Column, query Query) (string, error) {
	out := "SELECT rowid, "
	for i, col := range cols {
		out += columnName(col.Name)
		if i < len(cols)-1 {
			out += ", "
		}
	}
	return out + " FROM ds_" + strconv.Itoa(query.UID), nil
}

func makeFullQuery(cols []*Column, query Query) (string, []interface{}, error) {
	selectQuery, err := buildSelectQuery(cols, query)
	if err != nil {
		return "", nil, err
	}
	queryString, queryParameters, err := buildWhereQuery(cols, query)
	finalQuery := selectQuery
	if len(query.Filters) > 0 {
		finalQuery += " WHERE " + queryString
	}
	if query.Limit > 0 {
		finalQuery += " LIMIT " + strconv.Itoa(query.Limit)
	}
	finalQuery += " OFFSET " + strconv.Itoa(query.Offset)

	log.Println("StreamingQuery: ", finalQuery, queryParameters)
	return finalQuery, queryParameters, err
}
