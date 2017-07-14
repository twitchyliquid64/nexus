package util

import "database/sql"

// ColumnExists returns true if the specified column exists in the specified table.
func ColumnExists(tx *sql.Tx, colName, tableName string) (bool, error) {
	res, err := tx.Query("SELECT TableName, Name FROM __Column WHERE TableName = $1 AND Name = $2;", tableName, colName)
	if err != nil {
		return false, err
	}
	defer res.Close()

	if !res.Next() {
		return false, nil
	}
	return true, nil
}
