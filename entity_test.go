package gyr

import (
	"reflect"
	"testing"
)

type TestEntity struct {
	Name      string `gyr_column:"name"`
	Transient string
	Count     int `gyr_column:"count"`
}

func TestRegistry(t *testing.T) {
	metadata, err := getEntityMetadata[TestEntity]()
	if err.Error() != "unknown entity type" {
		t.Fail()
	}
	RegisterEntity[TestEntity](EntityMetadata{Table: "test_entity_table"})
	metadata, err = getEntityMetadata[TestEntity]()
	if err != nil {
		t.Fail()
	}
	if metadata.Table != "test_entity_table" {
		t.Fail()
	}
}

func TestDetectColumns(t *testing.T) {
	et := reflect.TypeFor[TestEntity]()
	columns := getColumnsFromType(et)
	if len(columns) != 2 {
		t.Fail()
	}
	if columns[0] != "name" || columns[1] != "count" {
		t.Fail()
	}
}

func TestCreateInsert(t *testing.T) {
	RegisterEntity[TestEntity](EntityMetadata{Table: "test_entity_table"})
	insertQuery, err := CreateInsertQuery[TestEntity]()
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if insertQuery != "insert into test_entity_table(name,count) values (?,?)" {
		t.Fail()
	}
}
