package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jxdones/stoat/internal/database"
)

// Databases returns the list of database names in the given path.
func Databases(ctx context.Context, db *sql.DB) ([]string, error) {
	if db == nil {
		return nil, database.ErrNoConnection
	}

	rows, err := db.QueryContext(ctx, `
		SELECT schema_name
  		FROM information_schema.schemata
  		WHERE schema_name NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
  		ORDER BY schema_name ASC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	databases := make([]string, 0)
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return nil, err
		}
		databases = append(databases, schema)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return databases, nil
}

// Tables returns the list of table names in the given database.
func Tables(ctx context.Context, db *sql.DB, databaseName string) ([]string, error) {
	if db == nil {
		return nil, database.ErrNoConnection
	}

	rows, err := db.QueryContext(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = ?
			AND table_type = 'BASE TABLE'
		ORDER BY table_name ASC;
	`, databaseName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := make([]string, 0)
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tables, nil
}

// Rows returns a page of rows for the given table.
// The result includes columns, rows, and pagination state (HasMore, NextAfter).
func Rows(ctx context.Context, db *sql.DB, target database.DatabaseTarget, page database.PageRequest) (database.PageResult, error) {
	if db == nil {
		return database.PageResult{}, database.ErrNoConnection
	}

	columnsInfo, err := columnInfoRows(ctx, db, target.Database, target.Table)
	if err != nil {
		return database.PageResult{}, err
	}

	if len(columnsInfo) == 0 {
		return database.PageResult{}, errors.New("table has no columns")
	}

	pageLimit := page.Limit
	if pageLimit <= 0 {
		pageLimit = 200
	}

	columnNames, selectColumns := buildPageColumns(columnsInfo)
	plan, err := buildRowsPagePlan(target.Database, target.Table, columnsInfo, columnNames, selectColumns, page, pageLimit)
	if err != nil {
		return database.PageResult{}, err
	}
	rows, hasMore, nextAfter, err := database.ScanRowsPageResult(ctx, db, plan, columnNames, pageLimit)
	if err != nil {
		return database.PageResult{}, err
	}
	return database.PageResult{
		Result: database.QueryResult{
			Columns: database.BuildOutputColumns(columnsInfo),
			Rows:    rows,
		},
		StartAfter: plan.AfterValue,
		HasMore:    hasMore,
		NextAfter:  nextAfter,
	}, nil
}

// Query executes a single SQL statement and returns a normalized result.
// For SELECT it returns columns and rows (capped at 1000); each row is a map from column name to string.
// For INSERT/UPDATE/DELETE it sets RowsAffected from MySQL's rowsAffected().
// Returns query result or an error if the query is invalid or the connection is lost.
func Query(ctx context.Context, db *sql.DB, query string) (database.QueryResult, error) {
	const maxRows = 1000
	const queryResultCap = 256

	if strings.TrimSpace(query) == "" {
		return database.QueryResult{}, database.ErrInvalidQuery
	}

	if db == nil {
		return database.QueryResult{}, database.ErrNoConnection
	}

	affectedRows := int64(0)
	columnNames := make([]string, 0)
	resultRows := make([]database.Row, 0, min(maxRows, queryResultCap))

	firstKeyword := database.FirstSQLKeyword(query)
	shouldReturnRows := firstKeyword == "SELECT" || firstKeyword == "EXPLAIN"
	if shouldReturnRows {
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return database.QueryResult{}, err
		}

		columnNames, err = rows.Columns()
		if err != nil {
			return database.QueryResult{}, err
		}
		values, targets := database.MakeScanBuffers(len(columnNames))

		for rows.Next() {
			if len(resultRows) >= maxRows {
				break
			}

			if err := rows.Scan(targets...); err != nil {
				rows.Close()
				return database.QueryResult{}, err
			}

			row := make(database.Row, len(columnNames))
			for i, name := range columnNames {
				row[name] = database.AsString(values[i])
			}
			resultRows = append(resultRows, row)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return database.QueryResult{}, err
		}
		if err := rows.Close(); err != nil {
			return database.QueryResult{}, err
		}
	} else {
		result, err := db.ExecContext(ctx, query)
		if err != nil {
			return database.QueryResult{}, err
		}
		affectedRows, err = result.RowsAffected()
		if err != nil {
			return database.QueryResult{}, err
		}
	}

	seen := make(map[string]struct{}, len(columnNames))
	orderedColumns := make([]string, 0, len(columnNames))
	for _, name := range columnNames {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		orderedColumns = append(orderedColumns, name)
	}

	columns := make([]database.Column, 0, len(orderedColumns))
	for i, column := range orderedColumns {
		columns = append(columns, database.Column{
			Key:      column,
			Title:    column,
			Type:     "text",
			MinWidth: database.ColumnMinWidth(len([]rune(column))),
			Order:    i + 1,
		})
	}

	return database.QueryResult{
		Columns:      columns,
		Rows:         resultRows,
		RowsAffected: affectedRows,
	}, nil
}

// Indexes returns the list of indexes for the given table.
func Indexes(ctx context.Context, db *sql.DB, target database.DatabaseTarget) ([]database.Index, error) {
	if db == nil {
		return nil, database.ErrNoConnection
	}

	rows, err := db.QueryContext(ctx, `
		SELECT
			index_name,
			column_name,
			non_unique
  		FROM information_schema.statistics                                  
  		WHERE table_schema = ?                                                                                                                                                                                         
    		AND table_name = ?  
  		ORDER BY index_name;   
	`, target.Database, target.Table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexByName := make(map[string]*database.Index)
	order := make([]string, 0)
	for rows.Next() {
		var (
			indexName  string
			columnName string
			nonUnique  bool
		)
		if err := rows.Scan(&indexName, &columnName, &nonUnique); err != nil {
			return nil, err
		}
		indexName = strings.TrimSpace(indexName)
		columnName = strings.TrimSpace(columnName)
		if indexName == "" || columnName == "" {
			continue
		}
		index, ok := indexByName[indexName]
		if !ok {
			index = &database.Index{
				Name:   indexName,
				Unique: !nonUnique,
			}
			indexByName[indexName] = index
			order = append(order, indexName)
		}
		index.Columns = append(index.Columns, columnName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]database.Index, 0, len(order))
	for _, name := range order {
		result = append(result, *indexByName[name])
	}
	return result, nil
}

// Constraints returns the list of constraints for the given table.
func Constraints(ctx context.Context, db *sql.DB, target database.DatabaseTarget) ([]database.Constraint, error) {
	if db == nil {
		return nil, database.ErrNoConnection
	}

	rows, err := db.QueryContext(ctx, `
		SELECT
			tc.constraint_name,
			tc.constraint_type,
			kcu.column_name                                                                                                                                                 
		FROM information_schema.table_constraints tc                  
		JOIN information_schema.key_column_usage kcu                                                                                                                                                                   
			ON kcu.constraint_name = tc.constraint_name           
			AND kcu.table_schema = tc.table_schema                                                                                                                                                                     
			AND kcu.table_name = tc.table_name                    
		WHERE tc.table_schema = ?                                                                                                                                                                                      
			AND tc.table_name = ?  
			AND tc.constraint_type IN ('PRIMARY KEY', 'UNIQUE')                                                                                                                                                          
		ORDER BY tc.constraint_type, tc.constraint_name, kcu.ordinal_position;
	`, target.Database, target.Table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	constraintsByName := make(map[string]*database.Constraint)
	order := make([]string, 0)
	for rows.Next() {
		var (
			constraintName string
			constraintType string
			columnName     string
		)
		if err := rows.Scan(&constraintName, &constraintType, &columnName); err != nil {
			return nil, err
		}
		constraintName = strings.TrimSpace(constraintName)
		columnName = strings.TrimSpace(columnName)
		if constraintName == "" || columnName == "" {
			continue
		}
		constraint, ok := constraintsByName[constraintName]
		if !ok {
			constraint = &database.Constraint{
				Name: constraintName,
				Type: constraintType,
			}
			constraintsByName[constraintName] = constraint
			order = append(order, constraintName)
		}
		constraint.Columns = append(constraint.Columns, columnName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]database.Constraint, 0, len(order))
	for _, name := range order {
		result = append(result, *constraintsByName[name])
	}
	return result, nil
}

// ForeignKeys returns the list of foreign keys for the given table.
func ForeignKeys(ctx context.Context, db *sql.DB, target database.DatabaseTarget) ([]database.ForeignKey, error) {
	if db == nil {
		return nil, database.ErrNoConnection
	}

	rows, err := db.QueryContext(ctx, `
		SELECT 
			rc.constraint_name,
			kcu.column_name,
			kcu.referenced_table_name,
			kcu.referenced_column_name,
			rc.update_rule,
			rc.delete_rule                                                                              
		FROM information_schema.referential_constraints rc                                                                               
		JOIN information_schema.key_column_usage kcu                                                                                                                                                                   
			ON kcu.constraint_name = rc.constraint_name           
			AND kcu.constraint_schema = rc.constraint_schema                                                                                                                                                           
		WHERE rc.constraint_schema = ?                            
			AND kcu.table_name = ?                                                                                                                                                                                       
		ORDER BY rc.constraint_name, kcu.ordinal_position;
	`, target.Database, target.Table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]database.ForeignKey, 0)
	for rows.Next() {
		var (
			constraintName       string
			columnName           string
			referencedTableName  string
			referencedColumnName string
			updateRule           string
			deleteRule           string
		)
		if err := rows.Scan(&constraintName, &columnName, &referencedTableName, &referencedColumnName, &updateRule, &deleteRule); err != nil {
			return nil, err
		}
		constraintName = strings.TrimSpace(constraintName)
		columnName = strings.TrimSpace(columnName)
		referencedTableName = strings.TrimSpace(referencedTableName)
		referencedColumnName = strings.TrimSpace(referencedColumnName)
		updateRule = strings.TrimSpace(updateRule)
		deleteRule = strings.TrimSpace(deleteRule)
		if constraintName == "" || columnName == "" || referencedTableName == "" || referencedColumnName == "" {
			continue
		}
		result = append(result, database.ForeignKey{
			Name:           constraintName,
			Column:         columnName,
			RefTable:       referencedTableName,
			RefColumn:      referencedColumnName,
			OnUpdateAction: updateRule,
			OnDeleteAction: deleteRule,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func columnInfoRows(ctx context.Context, db *sql.DB, schema, table string) ([]database.ColumnInfo, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			c.column_name,
			c.data_type,
			kcu.ordinal_position AS pk_order
		FROM information_schema.columns c
		LEFT JOIN information_schema.table_constraints tc
			ON tc.table_schema = c.table_schema
			AND tc.table_name = c.table_name
			AND tc.constraint_type = 'PRIMARY KEY'
		LEFT JOIN information_schema.key_column_usage kcu
			ON kcu.constraint_name = tc.constraint_name
			AND kcu.table_schema = c.table_schema
			AND kcu.table_name = c.table_name
			AND kcu.column_name = c.column_name
		WHERE c.table_schema = ?
			AND c.table_name = ?
		ORDER BY c.ordinal_position ASC;
	`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]database.ColumnInfo, 0)
	for rows.Next() {
		var (
			name     string
			typeName string
			pkOrder  sql.NullInt64
		)
		if err := rows.Scan(&name, &typeName, &pkOrder); err != nil {
			return nil, err
		}
		if strings.TrimSpace(name) == "" {
			continue
		}
		order := 0
		if pkOrder.Valid {
			order = int(pkOrder.Int64)
		}
		result = append(result, database.ColumnInfo{
			Name:    name,
			Type:    typeName,
			PKOrder: order,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// buildPageColumns returns column names and quoted SELECT column list for the given table columns.
func buildPageColumns(columnsInfo []database.ColumnInfo) ([]string, []string) {
	columnNames := make([]string, 0, len(columnsInfo))
	selectColumns := make([]string, 0, len(columnsInfo))
	for _, column := range columnsInfo {
		columnNames = append(columnNames, column.Name)
		selectColumns = append(selectColumns, database.QuoteIdentifierMySQL(column.Name))
	}
	return columnNames, selectColumns
}

// buildRowsPagePlan builds the SQL and metadata for one page of table rows.
// It chooses a strategy based on the table: integer primary key, or offset.
// Returns an error if the table cannot be inspected or the cursor is invalid.
func buildRowsPagePlan(schema, table string, columnsInfo []database.ColumnInfo, columnNames, selectColumns []string, page database.PageRequest, pageLimit int) (database.RowsPagePlan, error) {
	cursorAlias := database.UniqueCursorAlias(columnNames)
	primaryKeyColumns := database.OrderedPrimaryKeyColumns(columnsInfo)

	// Request one extra row to know if there is a next page (HasMore).
	limit := pageLimit + 1
	switch {
	case len(primaryKeyColumns) == 1 && database.HasIntegerAffinity(database.ColumnType(columnsInfo, primaryKeyColumns[0])):
		after, err := database.ParseCursor(page.After, "pk")
		if err != nil {
			return database.RowsPagePlan{}, err
		}
		primaryKey := database.QuoteIdentifierMySQL(primaryKeyColumns[0])
		where := ""
		if after > 0 {
			where = fmt.Sprintf("WHERE %s > %d", primaryKey, after)
		}
		return database.RowsPagePlan{
			Mode: database.PaginationByIntegerPK,
			Query: fmt.Sprintf(
				"SELECT %s AS %s, %s FROM %s.%s %s ORDER BY %s LIMIT %d;",
				primaryKey,
				database.QuoteIdentifierMySQL(cursorAlias),
				strings.Join(selectColumns, ", "),
				database.QuoteIdentifierMySQL(schema),
				database.QuoteIdentifierMySQL(table),
				where,
				primaryKey,
				limit,
			),
			ScanOffset: 1, // first column is the PK cursor
			AfterValue: after,
		}, nil
	default:
		offset, err := database.ParseCursor(page.After, "off")
		if err != nil {
			return database.RowsPagePlan{}, err
		}
		if offset < 0 {
			offset = 0
		}
		return database.RowsPagePlan{
			Mode: database.PaginationByOffset,
			Query: fmt.Sprintf(
				"SELECT %s FROM %s.%s ORDER BY %s LIMIT %d OFFSET %d;",
				strings.Join(selectColumns, ", "),
				database.QuoteIdentifierMySQL(schema),
				database.QuoteIdentifierMySQL(table),
				database.PrimaryKeyOrderExpr(primaryKeyColumns),
				limit,
				offset,
			),
			ScanOffset: 0, // no separate cursor column; offset is the cursor
			AfterValue: offset,
		}, nil
	}
}
