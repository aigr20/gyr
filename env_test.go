package gyr_test

import (
	"os"
	"testing"

	"github.com/aigr20/gyr"
)

func TestLoadEnvironment(t *testing.T) {
	gyr.EnvFile = "env_test_file"
	expectations := map[string]string{
		"VAR":            "32",
		"COMMENTED_LINE": "",
		"host":           "localhost",
	}
	err := gyr.LoadEnvironment()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	for name, expected := range expectations {
		if v := os.Getenv(name); v != expected {
			t.Logf("Expected %s to equal '%s'. Received '%s'\n", name, expected, v)
			t.FailNow()
		}
	}
}

func TestLoadEnvironmentDoesNotOverwrite(t *testing.T) {
	gyr.EnvFile = "env_test_file"
	os.Setenv("VAR", "exist")
	err := gyr.LoadEnvironment()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	if v := os.Getenv("VAR"); v != "exist" {
		t.Logf("Expected VAR to be 'exist' but received '%s'\n", v)
		t.FailNow()
	}
}
