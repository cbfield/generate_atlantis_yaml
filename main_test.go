package main

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"golang.org/x/exp/slices"
)

func prepEnv(t *testing.T) string {
	absPath, err := filepath.Abs("./test_data")

	if err != nil {
		t.Error("Cannot find test data")
	}

	t.Setenv("DIR",absPath)
	return absPath
}

func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "  ")
	return string(s)
}

func TestGetProjectsAndDependencies(t *testing.T) {
	absPath := prepEnv(t)

	gotProjects, gotDeps := getProjectsAndDependencies()

	expectedProjects := []string{absPath+"/project1", absPath+"/project2"}
	expectedDeps := map[string][]string{
		absPath:{},
		absPath+"/project1":{"../modules/module1"},
		absPath+"/project1/files":{},
		absPath+"/project2":{"../modules/module2"},
		absPath+"/modules":{},
		absPath+"/modules/module1":{"../module2"},
		absPath+"/modules/module2":{},
		absPath+"/modules/module2/files":{},
	}

	if !slices.Equal(gotProjects, expectedProjects) {
		t.Errorf("Got projects:\n%q\nExpected projects:\n%q", gotProjects, expectedProjects)
	}

	for key,value := range expectedDeps {
		gotValue, ok := gotDeps[key]
		if !ok {
			t.Errorf("dependencies key expected but not found: %s\n",key)
		}
		if !reflect.DeepEqual(value, gotValue) {
			t.Errorf("dependencies[%s]\nExpected:\n%s\nGot:\n%s\n",key,prettyPrint(value),prettyPrint(gotValue))
		}
	}

	for key,value := range gotDeps {
		expectedValue, ok := expectedDeps[key]
		if !ok {
			t.Errorf("dependencies key found but not expected: %s\n",key)
		}
		if !reflect.DeepEqual(value, expectedValue) {
			t.Errorf("dependencies[%s]\nExpected:\n%s\nGot:\n%s\n",key,prettyPrint(expectedValue),prettyPrint(value))
		}
	}
}

func TestReadAtlantisYaml(t *testing.T) {
	absPath, err := filepath.Abs("./test_data")

	if err != nil {
		t.Error("Cannot find test data")
	}

	t.Setenv("DIR",absPath)
	atlantisConfig := readAtlantisYaml()

	expectedConfig := AtlantisConfig{
		Automerge: true,
		DeleteSourceBranchOnMerge: true,
		ParallelApply: true,
		ParallelPlan: true,
		Version: 3,
	}

	if !reflect.DeepEqual(atlantisConfig, expectedConfig) {
		t.Errorf("Expected Atlantis Config:\n%s\n\nGot Atlantis Config:\n%s", prettyPrint(expectedConfig), prettyPrint(atlantisConfig))
	}
}

func TestAddProjectsToConfig(t *testing.T) {
	absPath := prepEnv(t)

	expectedConfig := AtlantisConfig{
		Automerge: true,
		DeleteSourceBranchOnMerge: true,
		ParallelApply: true,
		ParallelPlan: true,
		Projects: []ProjectConfig{
			{
				Name: "project1",
				Dir: "project1",
				Autoplan: AutoplanConfig{
					Enabled: true,
					WhenModified: []string{
						"**/*",
						"../modules/module1/**/*",
						"../modules/module2/**/*",
					},
				},
			},
			{
				Name: "project2",
				Dir: "project2",
				Autoplan: AutoplanConfig{
					Enabled: true,
					WhenModified: []string{
						"**/*",
						"../modules/module2/**/*",
					},
				},
			},
		},
		Version: 3,
	}

	projects := []string{absPath+"/project1", absPath+"/project2"}

	dependencies := map[string][]string{
		absPath+"/project1":{"../modules/module1"},
		absPath+"/project2":{"../modules/module2"},
		absPath+"/modules/module1":{"../module2"},
		absPath+"/modules/module2":{},
	}

	atlantisConfig := AtlantisConfig{
		Automerge: true,
		DeleteSourceBranchOnMerge: true,
		ParallelApply: true,
		ParallelPlan: true,
		Version: 3,
	}

	completeConfig := addProjectsToConfig(atlantisConfig, projects, dependencies)

	if !reflect.DeepEqual(expectedConfig, completeConfig) {
		t.Errorf("Expected Atlantis Config:\n%s\n\nGot Atlantis Config:\n%s\n",prettyPrint(expectedConfig),prettyPrint(completeConfig))
	}
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
