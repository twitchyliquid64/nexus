package datastore

import (
	"errors"
	"strconv"
	"strings"
)

// Query encapsulates all the information needed to perform a query on a datastore.
type Query struct {
	UID     int
	Filters []Filter
	Limit   int
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

//TODO: Sanitize/escape all non-alpha characters
func columnName(inName string) string {
	return strings.Replace(inName, " ", "_", -1) + "_"
}

//TODO: better type checking, sanitize conditional, support more datatypes than INT, UINT, STR (currently assume all non-ints are STR)
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
			queryString += columnName(col.Name) + " " + filter.Conditional + " $" + strconv.Itoa(i+1)
			if col.Datatype == INT || col.Datatype == UINT {
				v, _ := strconv.Atoi(filter.Val)
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
	out := "SELECT id(), "
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
		finalQuery += " WHERE " + queryString + ";"
	}
	return finalQuery, queryParameters, err
}
