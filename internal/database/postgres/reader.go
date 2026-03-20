package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jxdones/stoat/internal/database"
)

// paginationMode is the strategy for paginating a table (rowid, integer PK, or offset).
type paginationMode int

const (
	paginationByIntegerPK paginationMode = iota
	paginationByOffset
)

// columnInfo holds column metadata for one table column.
type columnInfo struct {
	Name    string
	Type    string
	PKOrder int
}

// rowsPagePlan holds the SQL and metadata for one page of table rows.
type rowsPagePlan struct {
	mode       paginationMode
	query      string
	scanOffset int
	afterValue int64
}

// Schemas returns the list of schemas in the PostgreSQL database.
func Schemas(ctx context.Context, db *sql.DB) ([]string, error) {
	if db == nil {
		return nil, database.ErrNoConnection
	}

	rows, err := db.QueryContext(ctx, `
		SELECT schema_name 
		FROM information_schema.schemata 
		WHERE schema_name NOT LIKE 'pg_%' 
			AND schema_name != 'information_schema'
		ORDER BY schema_name ASC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	schemas := make([]string, 0)
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return nil, err
		}
		schemas = append(schemas, schema)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return schemas, nil
}

// Tables returns the list of table names in the given schema.
func Tables(ctx context.Context, db *sql.DB, schema string) ([]string, error) {
	if db == nil {
		return nil, database.ErrNoConnection
	}

	rows, err := db.QueryContext(ctx, `
		SELECT table_name 
		FROM information_schema.tables
		WHERE table_schema = $1
			AND table_type = 'BASE TABLE'
		ORDER BY table_name ASC;
	`, schema)
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
	rows, hasMore, nextAfter, err := scanRowsPageResult(ctx, db, plan, columnNames, pageLimit)
	if err != nil {
		return database.PageResult{}, err
	}
	return database.PageResult{
		Result: database.QueryResult{
			Columns: buildOutputColumns(columnsInfo),
			Rows:    rows,
		},
		StartAfter: plan.afterValue,
		HasMore:    hasMore,
		NextAfter:  nextAfter,
	}, nil
}

// Query executes a single SQL statement and returns a normalized result.
// For SELECT it returns columns and rows (capped at 1000); each row is a map from column name to string.
// For INSERT/UPDATE/DELETE it sets RowsAffected from PostgreSQL's rowsAffected().
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

	firstKeyword := strings.Fields(strings.TrimSpace(strings.ToUpper(query)))[0]
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

// Constraints returns the list of constraints for the given table.
func Constraints(ctx context.Context, db *sql.DB, target database.DatabaseTarget) ([]database.Constraint, error) {
	if db == nil {
		return nil, database.ErrNoConnection
	}

	rows, err := db.QueryContext(ctx, `
		SELECT
			con.conname AS constraint_name,
			CASE con.contype WHEN 'p' THEN 'PRIMARY KEY' WHEN 'u' THEN 'UNIQUE' END AS constraint_type,
			a.attname AS column_name,
			NULL::text AS detail,
			CASE con.contype WHEN 'p' THEN 0 WHEN 'u' THEN 2 END AS sort_order
		FROM pg_catalog.pg_constraint con
		JOIN pg_catalog.pg_class c ON c.oid = con.conrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_catalog.pg_attribute a
			ON a.attrelid = con.conrelid AND a.attnum = ANY(con.conkey)
		WHERE c.relname = $1 AND n.nspname = $2
			AND con.contype IN ('p', 'u')

		UNION ALL

		SELECT
			'NOT NULL ' || a.attname,
			'NOT NULL',
			a.attname,
			NULL::text,
			1
		FROM pg_catalog.pg_attribute a
		JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relname = $1 AND n.nspname = $2
			AND a.attnotnull = true AND a.attnum > 0 AND NOT a.attisdropped

		UNION ALL

		SELECT
			'DEFAULT ' || a.attname,
			'DEFAULT',
			a.attname,
			pg_get_expr(ad.adbin, ad.adrelid),
			1
		FROM pg_catalog.pg_attribute a
		JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_catalog.pg_attrdef ad ON ad.adrelid = a.attrelid AND ad.adnum = a.attnum
		WHERE c.relname = $1 AND n.nspname = $2
			AND a.attnum > 0 AND NOT a.attisdropped
		ORDER BY sort_order, constraint_name, column_name;
	`, target.Table, target.Database)
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
			detail         sql.NullString
			sortOrder      int
		)
		if err := rows.Scan(&constraintName, &constraintType, &columnName, &detail, &sortOrder); err != nil {
			return nil, err
		}
		constraintName = strings.TrimSpace(constraintName)
		columnName = strings.TrimSpace(columnName)
		if constraintName == "" || columnName == "" {
			continue
		}
		c, ok := constraintsByName[constraintName]
		if !ok {
			c = &database.Constraint{
				Name: constraintName,
				Type: constraintType,
			}
			if detail.Valid && strings.TrimSpace(detail.String) != "" {
				c.Detail = detail.String
			}
			constraintsByName[constraintName] = c
			order = append(order, constraintName)
		}
		c.Columns = append(c.Columns, columnName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]database.Constraint, 0, len(order))
	for _, name := range order {
		if c, ok := constraintsByName[name]; ok {
			result = append(result, *c)
		}
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
			c.conname AS constraint_name,
			a.attname AS column_name,
			ref.relname AS ref_table,
			ra.attname AS ref_column,
			c.confupdtype,
			c.confdeltype
		FROM pg_catalog.pg_constraint c
		JOIN pg_catalog.pg_class cl ON cl.oid = c.conrelid
		JOIN pg_catalog.pg_class ref ON ref.oid = c.confrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = cl.relnamespace
		JOIN pg_catalog.pg_attribute a
			ON a.attrelid = c.conrelid AND a.attnum = ANY(c.conkey)
		JOIN pg_catalog.pg_attribute ra
			ON ra.attrelid = c.confrelid AND ra.attnum = ANY(c.confkey)
		WHERE cl.relname = $1
			AND n.nspname = $2
			AND c.contype = 'f'
		ORDER BY c.conname;
	`, target.Table, target.Database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]database.ForeignKey, 0)
	for rows.Next() {
		var (
			name           string
			column         string
			refTable       string
			refColumn      string
			onUpdateAction string
			onDeleteAction string
		)
		if err := rows.Scan(&name, &column, &refTable, &refColumn, &onUpdateAction, &onDeleteAction); err != nil {
			return nil, err
		}

		name = strings.TrimSpace(name)
		column = strings.TrimSpace(column)
		refTable = strings.TrimSpace(refTable)
		refColumn = strings.TrimSpace(refColumn)
		onUpdateAction = foreignKeyAction(strings.TrimSpace(onUpdateAction))
		onDeleteAction = foreignKeyAction(strings.TrimSpace(onDeleteAction))

		if column == "" || refTable == "" || refColumn == "" {
			continue
		}

		result = append(result, database.ForeignKey{
			Name:           name,
			Column:         column,
			RefTable:       refTable,
			RefColumn:      refColumn,
			OnUpdateAction: strings.TrimSpace(onUpdateAction),
			OnDeleteAction: strings.TrimSpace(onDeleteAction),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// Indexes returns the list of indexes for the given table.
func Indexes(ctx context.Context, db *sql.DB, target database.DatabaseTarget) ([]database.Index, error) {
	if db == nil {
		return nil, database.ErrNoConnection
	}

	rows, err := db.QueryContext(ctx, `
		SELECT
			ic.relname AS index_name,
			ix.indisunique AS is_unique,
			a.attname AS column_name,
			a.attnum
		FROM pg_catalog.pg_index ix
		JOIN pg_catalog.pg_class c  ON c.oid = ix.indrelid
		JOIN pg_catalog.pg_class ic ON ic.oid = ix.indexrelid
		JOIN pg_catalog.pg_attribute a
			ON a.attrelid = c.oid
			AND a.attnum = ANY(ix.indkey)
			AND a.attnum > 0
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relname = $1
			AND n.nspname = $2
		ORDER BY ic.relname, a.attnum ASC;
	`, target.Table, target.Database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexByName := make(map[string]*database.Index)
	order := make([]string, 0)
	for rows.Next() {
		var (
			indexName  string
			isUnique   bool
			columnName string
			attnum     int64
		)
		if err := rows.Scan(&indexName, &isUnique, &columnName, &attnum); err != nil {
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
				Unique: isUnique,
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

// columnInfoRows returns the list of columns for the given table.
func columnInfoRows(ctx context.Context, db *sql.DB, schema, table string) ([]columnInfo, error) {
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
		WHERE c.table_schema = $1
			AND c.table_name = $2
		ORDER BY c.ordinal_position ASC;
	`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]columnInfo, 0)
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
		result = append(result, columnInfo{
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
func buildPageColumns(columnsInfo []columnInfo) ([]string, []string) {
	columnNames := make([]string, 0, len(columnsInfo))
	selectColumns := make([]string, 0, len(columnsInfo))
	for _, column := range columnsInfo {
		columnNames = append(columnNames, column.Name)
		selectColumns = append(selectColumns, database.QuoteIdentifier(column.Name))
	}
	return columnNames, selectColumns
}

// buildRowsPagePlan builds the SQL and metadata for one page of table rows.
// It chooses a strategy based on the table: integer primary key, or offset.
// Returns an error if the table cannot be inspected or the cursor is invalid.
func buildRowsPagePlan(schema, table string, columnsInfo []columnInfo, columnNames, selectColumns []string, page database.PageRequest, pageLimit int) (rowsPagePlan, error) {
	cursorAlias := database.UniqueCursorAlias(columnNames)
	primaryKeyColumns := orderedPrimaryKeyColumns(columnsInfo)

	// Request one extra row to know if there is a next page (HasMore).
	limit := pageLimit + 1
	switch {
	case len(primaryKeyColumns) == 1 && database.HasIntegerAffinity(columnType(columnsInfo, primaryKeyColumns[0])):
		after, err := database.ParseCursor(page.After, "pk")
		if err != nil {
			return rowsPagePlan{}, err
		}
		primaryKey := database.QuoteIdentifier(primaryKeyColumns[0])
		where := ""
		if after > 0 {
			where = fmt.Sprintf("WHERE %s > %d", primaryKey, after)
		}
		return rowsPagePlan{
			mode: paginationByIntegerPK,
			query: fmt.Sprintf(
				"SELECT %s AS %s, %s FROM %s.%s %s ORDER BY %s LIMIT %d;",
				primaryKey,
				database.QuoteIdentifier(cursorAlias),
				strings.Join(selectColumns, ", "),
				database.QuoteIdentifier(schema),
				database.QuoteIdentifier(table),
				where,
				primaryKey,
				limit,
			),
			scanOffset: 1, // first column is the PK cursor
			afterValue: after,
		}, nil
	default:
		offset, err := database.ParseCursor(page.After, "off")
		if err != nil {
			return rowsPagePlan{}, err
		}
		if offset < 0 {
			offset = 0
		}
		return rowsPagePlan{
			mode: paginationByOffset,
			query: fmt.Sprintf(
				"SELECT %s FROM %s.%s ORDER BY %s LIMIT %d OFFSET %d;",
				strings.Join(selectColumns, ", "),
				database.QuoteIdentifier(schema),
				database.QuoteIdentifier(table),
				database.PrimaryKeyOrderExpr(primaryKeyColumns),
				limit,
				offset,
			),
			scanOffset: 0, // no separate cursor column; offset is the cursor
			afterValue: offset,
		}, nil
	}
}

// scanRowsPageResult executes the plan's query and returns up to pageLimit rows.
// The boolean reports whether more rows exist; the string is an opaque cursor for the next page.
func scanRowsPageResult(ctx context.Context, db *sql.DB, plan rowsPagePlan, columnNames []string, pageLimit int) ([]database.Row, bool, string, error) {
	if db == nil {
		return nil, false, "", database.ErrNoConnection
	}

	rows, err := db.QueryContext(ctx, plan.query)
	if err != nil {
		return nil, false, "", err
	}
	defer rows.Close()

	values, targets := database.MakeScanBuffers(len(columnNames) + plan.scanOffset) // allocate an extra slot for the cursor column if needed
	outputRows := make([]database.Row, 0, pageLimit)
	hasMore := false
	nextAfter := ""

	for rows.Next() {
		if err := rows.Scan(targets...); err != nil {
			return nil, false, "", err
		}
		if len(outputRows) >= pageLimit {
			hasMore = true
			break
		}
		switch plan.mode {
		case paginationByOffset:
			nextAfter = database.FormatCursor("off", plan.afterValue+int64(len(outputRows))+database.OffsetCursorSkipCurrentRow)
		case paginationByIntegerPK:
			nextAfter = database.FormatCursor("pk", database.AsInt64(values[0]))
		}
		row := make(database.Row, len(columnNames))
		for i, name := range columnNames {
			row[name] = database.AsString(values[i+plan.scanOffset])
		}
		outputRows = append(outputRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, false, "", err
	}
	return outputRows, hasMore, nextAfter, nil
}

// orderedPrimaryKeyColumns returns the primary key column names in table order.
// Non-PK columns and columns with empty names are ignored.
func orderedPrimaryKeyColumns(columnsInfo []columnInfo) []string {
	orderToColumn := make(map[int]string, len(columnsInfo))
	for _, column := range columnsInfo {
		if column.PKOrder <= 0 {
			continue
		}
		name := strings.TrimSpace(column.Name)
		if name == "" {
			continue
		}
		orderToColumn[column.PKOrder] = name
	}

	ordered := make([]int, 0, len(orderToColumn))
	for order := range orderToColumn {
		ordered = append(ordered, order)
	}
	sort.Ints(ordered)
	orderedColumns := make([]string, 0, len(ordered))
	for _, order := range ordered {
		orderedColumns = append(orderedColumns, orderToColumn[order])
	}

	return orderedColumns
}

// columnType returns the declared type of the given column.
func columnType(columnsInfo []columnInfo, name string) string {
	for _, column := range columnsInfo {
		if strings.EqualFold(strings.TrimSpace(column.Name), name) {
			return strings.TrimSpace(column.Type)
		}
	}
	return ""
}

// buildOutputColumns builds the display column list for the given table columns.
func buildOutputColumns(columnsInfo []columnInfo) []database.Column {
	columns := make([]database.Column, 0, len(columnsInfo))
	for i, column := range columnsInfo {
		declaredType := strings.TrimSpace(column.Type)
		if declaredType == "" {
			declaredType = "text"
		}
		columns = append(columns, database.Column{
			Key:      column.Name,
			Title:    column.Name,
			Type:     strings.ToLower(declaredType),
			MinWidth: database.ColumnMinWidth(len([]rune(column.Name))),
			Order:    i + 1, // preserve table column order (1-based for display)
		})
	}
	return columns
}

// foreignKeyAction converts the PostgreSQL foreign key action to a readable string.
func foreignKeyAction(action string) string {
	switch action {
	case "a":
		return "NO ACTION"
	case "r":
		return "RESTRICT"
	case "c":
		return "CASCADE"
	case "n":
		return "SET NULL"
	case "d":
		return "SET DEFAULT"
	default:
		return action
	}
}
