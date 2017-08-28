package datastore

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
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
	Val         interface{}
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

func coerceValueForColDatatype(datatype Datatype, val interface{}) (interface{}, error) {
	switch datatype {
	case TIME:
		switch v := val.(type) {
		case time.Time:
			return v, nil
		case int:
			return time.Unix(int64(v/1000), int64(v%1e9)), nil
		case int64:
			return time.Unix(int64(v/1000), int64(v%1e9)), nil
		case uint64:
			return time.Unix(int64(v/1000), int64(v%1e9)), nil
		case float32:
			return time.Unix(int64(v/1000), 0), nil
		case float64:
			return time.Unix(int64(v/1000), 0), nil
		case string:
			s, strErr := strconv.Atoi(v)
			if strErr != nil {
				return nil, fmt.Errorf("could not convert string for time column: %s", strErr.Error())
			}
			return time.Unix(int64(s/1000), int64(s%1e9)), nil
		}
	case INT, UINT:
		switch v := val.(type) {
		case time.Time:
			return v.Unix(), nil
		case int, int16, int32, int64, uint, uint16, uint32, uint64:
			return v, nil
		case float32:
			return int64(v), nil
		case float64:
			return int64(v), nil
		case string:
			s, strErr := strconv.Atoi(v)
			if strErr != nil {
				return nil, fmt.Errorf("could not convert string for integer column: %s", strErr.Error())
			}
			return s, nil
		}
	case STR:
		switch v := val.(type) {
		case time.Time:
			return v.String(), nil
		case int, int16, int32, int64, uint, uint16, uint32, uint64, float32, float64:
			return fmt.Sprint(v), nil
		case string:
			return v, nil
		}
	case BLOB:
		switch v := val.(type) {
		case time.Time:
			return v.String(), nil
		case int, int16, int32, int64, uint, uint16, uint32, uint64, float32, float64:
			return fmt.Sprint(v), nil
		case string:
			return []byte(v), nil
		case []byte:
			return v, nil
		}
	case FLOAT:
		switch v := val.(type) {
		case time.Time:
			return nil, errors.New("cannot co-erce float to time.Time")
		case int:
			return float64(v), nil
		case int64:
			return float64(v), nil
		case float32:
			return float64(v), nil
		case float64:
			return v, nil
		case string:
			return v, nil
		case []byte:
			return string(v), nil
		}
	}
	return nil, errors.New("unrecognised column datatype")
}

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
			val, err := coerceValueForColDatatype(col.Datatype, filter.Val)
			if err != nil {
				return "", nil, err
			}
			queryParameters = append(queryParameters, val)

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
		finalQuery += " OFFSET " + strconv.Itoa(query.Offset)
	}

	log.Println("StreamingQuery: ", finalQuery, queryParameters)
	return finalQuery, queryParameters, err
}
