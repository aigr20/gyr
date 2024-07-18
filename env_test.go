package gyr_test

import (
	"os"
	"testing"

	"github.com/aigr20/gyr"
)

func TestLoadEnvironment(t *testing.T) {
	gyr.EnvFile = "env_test_file"
	err := gyr.LoadEnvironment()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	if v := os.Getenv("VAR"); v != "32" {
		t.Logf("Expected VAR to be '32'. Received '%s'\n", v)
		t.FailNow()
	}
	if v := os.Getenv("COMMENTED_LINE"); v != "" {
		t.Logf("Expected COMMENTED_LINE to be empty. Received '%s'\n", v)
		t.FailNow()
	}
	if v := os.Getenv("host"); v != "localhost" {
		t.Logf("Expected host to be 'localhost'. Received '%s'\n", v)
		t.FailNow()
	}
}
