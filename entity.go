package gyr

import (
	"errors"
	"reflect"
	"strings"
)

type EntityMetadata struct {
	Table string
	// Is overwritten by RegisterEntity if a field with a gyr_column tag is detected in the struct being registered
	Columns []string
}

var (
	entityRegistry = make(map[reflect.Type]EntityMetadata)
)

const (
	gyr_column_tag = "gyr_column"
)

func RegisterEntity[EntityType any](metadata EntityMetadata) {
	entityType := reflect.TypeFor[EntityType]()
	if detectedColumns := getColumnsFromType(entityType); len(detectedColumns) > 0 {
		metadata.Columns = detectedColumns
	}
	entityRegistry[entityType] = metadata
}

func CreateSelectAllQuery[EntityType any]() (string, error) {
	metadata, err := getEntityMetadata[EntityType]()
	if err != nil {
		return "", err
	}
	query := strings.Builder{}
	query.WriteString("select * from ")
	query.WriteString(metadata.Table)
	return query.String(), nil
}

func CreateInsertQuery[EntityType any]() (string, error) {
	metadata, err := getEntityMetadata[EntityType]()
	if err != nil {
		return "", err
	}
	query := strings.Builder{}
	query.WriteString("insert into ")
	query.WriteString(metadata.Table)
	query.WriteRune('(')

	maxColumnIndex := len(metadata.Columns) - 1
	for i, columnName := range metadata.Columns {
		query.WriteString(columnName)
		if i < maxColumnIndex {
			query.WriteRune(',')
		}
	}

	query.WriteRune(')')
	query.WriteString(" values (")
	query.WriteString(nVars(len(metadata.Columns)))
	query.WriteRune(')')

	return query.String(), nil
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
