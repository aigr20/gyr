package gyr

import "testing"

func TestRemoveAlreadyMigratedPaths(t *testing.T) {
	paths := []string{"0.0.1_init.sql", "0.0.3_insert.sql", "0.0.2_alter.sql"}
	lastVersion := "0.0.2"
	pathsRemoved := removeAlreadyMigratedPaths(paths, lastVersion)
	if len(pathsRemoved) != 1 && pathsRemoved[0] != paths[1] {
		t.Logf("pathsRemoved contained %+v\n", pathsRemoved)
		t.FailNow()
	}
}
