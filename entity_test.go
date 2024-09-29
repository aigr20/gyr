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

func TestCreateSelectAll(t *testing.T) {
	RegisterEntity[TestEntity](EntityMetadata{Table: "test_entity_table"})
	query, err := CreateSelectAllQuery[TestEntity]()
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if query != "select name, count from test_entity_table" {
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
	if insertQuery != "insert into test_entity_table (name, count) values (?,?)" {
		t.Fail()
	}
}

func TestMultiInsertBuilder(t *testing.T) {
	RegisterEntity[TestEntity](EntityMetadata{Table: "test_entity_table"})
	query := NewQuery[TestEntity]().Insert([]string{"name", "count"}).AddValue().AddValue().AddValue().Query()
	if query != "insert into test_entity_table (name, count) values (?,?),(?,?),(?,?)" {
		t.Fail()
	}
}

func TestSelectBuilderPanics(t *testing.T) {
	RegisterEntity[TestEntity](EntityMetadata{Table: "test_entity_table"})
	qb := NewQuery[TestEntity]()
	qb.SelectAll()
	defer func() {
		if nilIfNoRecover := recover(); nilIfNoRecover == nil {
			t.Fail()
		}
	}()
	qb.SelectAll()
}

func TestSelectBuilderWhere(t *testing.T) {
	RegisterEntity[TestEntity](EntityMetadata{Table: "test_entity_table"})
	qb := NewQuery[TestEntity]()
	query := qb.SelectAll().Where("name").EqualsValue("kalle karlsson").And("count").EqualsVar().Query()
	if query != "select name, count from test_entity_table where name = 'kalle karlsson' and count = ?" {
		t.Fail()
	}
}

func TestRegisterEntityPanics(t *testing.T) {
	defer func() {
		if recoveredError := recover(); recoveredError != "no table defined for entity TestEntity" {
			t.Fail()
		}
	}()
	RegisterEntity[TestEntity](EntityMetadata{})
}
