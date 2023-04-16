package main

import (
	"path/filepath"
	"testing"

	"golang.org/x/exp/slices"
)

func TestGetProjectsAndDependencies(t *testing.T) {
	absPath, err := filepath.Abs("./test_data")
	if err != nil {
		t.Error("Cannot find test data")
	}
	t.Setenv("DIR",absPath)
	got_projects, got_deps := getProjectsAndDependencies()

	expected_projects := []string{absPath+"/project1", absPath+"/project2"}
	expected_deps := map[string][]string{
		absPath+"/project1":[]string{"../modules/module1"},
		absPath+"/project2":[]string{"../modules/module2"},
	}

	if !slices.Equal(got_projects, expected_projects) {
		t.Errorf("Got projects:\n%q\nExpected projects:\n%q", got_projects, expected_projects)
	}

	for k,v := range expected_deps {
		got_v, ok := got_deps[k]
		if !ok {
			t.Errorf("dependencies key expected but not found: %s\n",k)
		}
		if !slices.Equal(v, got_v) {
			t.Errorf("dependencies[%s]\nExpected:%s\nGot:%s\n",k,v,got_v)
		}
	}
}

func TestReadAtlantisYaml(t *testing.T) {
}

func TestAddProjectsToConfig(t *testing.T) {
}

func TestFileExists(t *testing.T) {
}

func TestMakeProjectConfig(t *testing.T) {
}

func TestGetWhenModifiedPaths(t *testing.T) {
}

func TestCleanPaths(t *testing.T) {
}

func TestUnique(t *testing.T) {
}

func TestWriteAtlantisYaml(t *testing.T) {
}

func TestMain(t *testing.T) {
}
