package gyr

import (
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// Metadata for a registered entity. Table has to be set, everything else is optional. Columns can be used instead of gyr_column tags on struct fields.
type EntityMetadata struct {
	Table string
	// Is overwritten by RegisterEntity if a field with a gyr_column tag is detected in the struct being registered
	Columns []string
}

const (
	queryType           = 1
	queryColumns        = 1 << 1
	queryIsInConditions = 1 << 2
	queryHasValueAdded  = 1 << 3
)

type BaseQueryBuilder interface {
	// Get the SQL Query in its current state from the builder
	Query() string
}

type QueryBuilder[EntityType any] struct {
	sb             *strings.Builder
	entityMetadata EntityMetadata
	fieldsSet      int
}

type SelectBuilder interface {
	BaseQueryBuilder
	// Start adding WHERE-conditions to your query.
	Where(string) WhereBuilder
}

type InsertBuilder interface {
	BaseQueryBuilder
	// Add a set of values to the INSERT-query
	AddValue() InsertBuilder
}

type WhereBuilder interface {
	BaseQueryBuilder
	// Equals condition with a SQL template variable
	EqualsVar() WhereBuilder
	// Equals a set value
	EqualsValue(any) WhereBuilder
	And(string) WhereBuilder
	Or(string) WhereBuilder
}

var (
	entityRegistry = make(map[reflect.Type]EntityMetadata)
)

const (
	gyr_column_tag = "gyr_column"
)

// Get a query builder instance. The entity must be registered using RegisterEntity.
func NewQuery[EntityType any]() *QueryBuilder[EntityType] {
	metadata, err := getEntityMetadata[EntityType]()
	if err != nil {
		return nil
	}
	return &QueryBuilder[EntityType]{
		sb:             &strings.Builder{},
		entityMetadata: metadata,
	}
}

func (qb *QueryBuilder[EntityType]) Query() string {
	return qb.sb.String()
}

func (qb *QueryBuilder[EntityType]) SelectAll() SelectBuilder {
	return qb.Select(qb.entityMetadata.Columns)
}

func (qb *QueryBuilder[EntityType]) Select(columns []string) SelectBuilder {
	if qb.fieldsSet&queryType > 0 {
		panic("query type already set")
	}
	for _, column := range columns {
		if !qb.hasColumn(column) {
			panic("Unknown column: " + column)
		}
	}

	qb.sb.WriteString("select ")
	qb.sb.WriteString(strings.Join(columns, ", "))
	qb.sb.WriteString(" from ")
	qb.sb.WriteString(qb.entityMetadata.Table)
	qb.fieldsSet |= queryType
	return qb
}

// Create an INSERT-query using all registered columns.
func (qb *QueryBuilder[EntityType]) InsertAll() InsertBuilder {
	return qb.Insert(qb.entityMetadata.Columns)
}

// Create an INSERT-query with a subset of all columns.
func (qb *QueryBuilder[EntityType]) Insert(columns []string) InsertBuilder {
	if qb.fieldsSet&queryType > 0 {
		panic("query type already set")
	}

	for _, column := range columns {
		if !qb.hasColumn(column) {
			panic("Unknown column: " + column)
		}
	}

	qb.sb.WriteString("insert into ")
	qb.sb.WriteString(qb.entityMetadata.Table)
	qb.sb.WriteString(" (")
	qb.sb.WriteString(strings.Join(columns, ", "))
	qb.sb.WriteString(") values ")
	qb.entityMetadata.Columns = columns
	return qb
}

func (qb *QueryBuilder[EntityType]) AddValue() InsertBuilder {
	if qb.fieldsSet&queryHasValueAdded > 0 {
		qb.sb.WriteRune(',')
	}
	qb.sb.WriteRune('(')
	qb.sb.WriteString(nVars(len(qb.entityMetadata.Columns)))
	qb.sb.WriteRune(')')
	qb.fieldsSet |= queryHasValueAdded
	return qb
}

func (qb *QueryBuilder[EntityType]) Where(column string) WhereBuilder {
	if qb.fieldsSet&queryType == 0 {
		panic("no query type set")
	}
	if !qb.hasColumn(column) {
		panic("Unknown column: " + column)
	}

	qb.sb.WriteString(" where ")
	qb.sb.WriteString(column)
	qb.fieldsSet |= queryIsInConditions
	return qb
}

func (qb *QueryBuilder[EntityType]) And(column string) WhereBuilder {
	if qb.fieldsSet&queryIsInConditions == 0 {
		panic("QueryBuilder is not in conditions phase")
	}
	if !qb.hasColumn(column) {
		panic("Unknown column: " + column)
	}

	qb.sb.WriteString(" and ")
	qb.sb.WriteString(column)
	return qb
}

func (qb *QueryBuilder[EntityType]) EqualsVar() WhereBuilder {
	return qb.EqualsValue("?")
}

func (qb *QueryBuilder[EntityType]) EqualsValue(value any) WhereBuilder {
	if qb.fieldsSet&queryIsInConditions == 0 {
		panic("QueryBuilder is not in conditions phase")
	}
	qb.sb.WriteString(" = ")
	writeBasedOnType(qb.sb, value)
	return qb
}

func (qb *QueryBuilder[EntityType]) Or(column string) WhereBuilder {
	if qb.fieldsSet&queryIsInConditions == 0 {
		panic("QueryBuilder is not in conditions phase")
	}
	if !qb.hasColumn(column) {
		panic("Unknown column: " + column)
	}

	qb.sb.WriteString(" or ")
	qb.sb.WriteString(column)
	return qb
}

func (qb QueryBuilder[EntityType]) hasColumn(columnName string) bool {
	return slices.Contains(qb.entityMetadata.Columns, columnName)
}

// Register an entity in the Gyr entity registry. Needs to be done in order to use the SQL helper methods in the Gyr library.
func RegisterEntity[EntityType any](metadata EntityMetadata) {
	entityType := reflect.TypeFor[EntityType]()

	if metadata.Table == "" {
		panic("no table defined for entity " + entityType.Name())
	}
	if detectedColumns := getColumnsFromType(entityType); len(detectedColumns) > 0 {
		metadata.Columns = detectedColumns
	}
	entityRegistry[entityType] = metadata
}

// Helper method for creating a SELECT * query without any conditions
func CreateSelectAllQuery[EntityType any]() (string, error) {
	query := NewQuery[EntityType]()
	if query == nil {
		return "", errors.New("unknown entity type")
	}
	return query.SelectAll().Query(), nil
}

// Helper method for creating an insert for a single instance of an entity using all columns
func CreateInsertQuery[EntityType any]() (string, error) {
	query := NewQuery[EntityType]()
	if query == nil {
		return "", errors.New("unknown entity type")
	}
	return query.InsertAll().AddValue().Query(), nil
}

func getColumnsFromType(entityType reflect.Type) []string {
	columns := make([]string, 0)
	fieldCount := entityType.NumField()
	for i := 0; i < fieldCount; i++ {
		field := entityType.Field(i)
		if columnName, hasTag := field.Tag.Lookup(gyr_column_tag); hasTag {
			columns = append(columns, columnName)
		}
	}

	return columns
}

func getEntityMetadata[EntityType any]() (EntityMetadata, error) {
	entityType := reflect.TypeFor[EntityType]()
	metadata, ok := entityRegistry[entityType]
	if !ok {
		return metadata, errors.New("unknown entity type")
	}
	return metadata, nil
}

func nVars(n int) string {
	return strings.Repeat("?,", n)[:(n*2)-1]
}

func writeBasedOnType(sb *strings.Builder, value any) {
	switch v := value.(type) {
	case string:
		if v == "?" {
			sb.WriteString(v)
		} else {
			sb.WriteRune('\'')
			sb.WriteString(v)
			sb.WriteRune('\'')
		}
	case int:
		sb.WriteString(strconv.Itoa(v))
	}
}
