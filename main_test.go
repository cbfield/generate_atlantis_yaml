package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/exp/slices"
)

func prepEnv(t *testing.T) string {
	absPath, err := filepath.Abs("./test_data")

	if err != nil {
		t.Error("Cannot find test data")
	}

	t.Setenv("DIR", absPath)
	return absPath
}

func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "  ")
	return string(s)
}

func revertAtlantisYaml(absPath string) {
	baseContent := "automerge: true\n" +
		"delete_source_branch_on_merge: true\n" +
		"parallel_apply: true\n" +
		"parallel_plan: true\n" +
		"version: 3\n"

	err := ioutil.WriteFile(absPath+"/atlantis.yaml", []byte(baseContent), 0)

	if err != nil {
		log.Fatal(err)
	}
}

func TestGetProjectsAndDependencies(t *testing.T) {
	absPath := prepEnv(t)

	gotProjects, gotDeps := getProjectsAndDependencies()

	expectedProjects := []string{absPath + "/project1", absPath + "/project2"}
	expectedDeps := map[string][]string{
		absPath:                            {},
		absPath + "/project1":              {"../modules/module1"},
		absPath + "/project1/files":        {},
		absPath + "/project2":              {"../modules/module2"},
		absPath + "/modules":               {},
		absPath + "/modules/module1":       {"../module2"},
		absPath + "/modules/module2":       {},
		absPath + "/modules/module2/files": {},
	}

	if !slices.Equal(gotProjects, expectedProjects) {
		t.Errorf("Got projects:\n%q\nExpected projects:\n%q", gotProjects, expectedProjects)
	}

	for key, value := range expectedDeps {
		gotValue, ok := gotDeps[key]
		if !ok {
			t.Errorf("dependencies key expected but not found: %q\n", key)
		}
		if !reflect.DeepEqual(value, gotValue) {
			t.Errorf(strings.Join([]string{
				"Evaluating calculated dependencies for dir %q",
				"Expected:\n%s\n",
				"Got:\n%s",
			}, "\n"), key, prettyPrint(value), prettyPrint(gotValue))
		}
	}

	for key, value := range gotDeps {
		expectedValue, ok := expectedDeps[key]
		if !ok {
			t.Errorf("dependencies key found but not expected: %q\n", key)
		}
		if !reflect.DeepEqual(value, expectedValue) {
			t.Errorf(strings.Join([]string{
				"Evaluating expected dependencies for dir %q",
				"Expected:\n%s\n",
				"Got:\n%s",
			}, "\n"), key, prettyPrint(expectedValue), prettyPrint(value))
		}
	}
}

func TestReadAtlantisYaml(t *testing.T) {
	absPath, err := filepath.Abs("./test_data")

	if err != nil {
		t.Error("Cannot find test data")
	}

	t.Setenv("DIR", absPath)
	atlantisConfig := readAtlantisYaml()

	expectedConfig := AtlantisConfig{
		Automerge:                 true,
		DeleteSourceBranchOnMerge: true,
		ParallelApply:             true,
		ParallelPlan:              true,
		Version:                   3,
	}

	if !reflect.DeepEqual(atlantisConfig, expectedConfig) {
		t.Errorf(strings.Join([]string{
			"Expected atlantis config:\n%s\n",
			"Got atlantis config:\n%s",
		}, "\n"), prettyPrint(expectedConfig), prettyPrint(atlantisConfig))
	}
}

func TestAddProjectsToConfig(t *testing.T) {
	absPath := prepEnv(t)

	expectedConfig := AtlantisConfig{
		Automerge:                 true,
		DeleteSourceBranchOnMerge: true,
		ParallelApply:             true,
		ParallelPlan:              true,
		Projects: []ProjectConfig{
			{
				Name: "project1",
				Dir:  "project1",
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
				Dir:  "project2",
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

	projects := []string{absPath + "/project1", absPath + "/project2"}

	dependencies := map[string][]string{
		absPath + "/project1":        {"../modules/module1"},
		absPath + "/project2":        {"../modules/module2"},
		absPath + "/modules/module1": {"../module2"},
		absPath + "/modules/module2": {},
	}

	atlantisConfig := AtlantisConfig{
		Automerge:                 true,
		DeleteSourceBranchOnMerge: true,
		ParallelApply:             true,
		ParallelPlan:              true,
		Version:                   3,
	}

	completeConfig := addProjectsToConfig(atlantisConfig, projects, dependencies)

	if !reflect.DeepEqual(expectedConfig, completeConfig) {
		t.Errorf(strings.Join([]string{
			"Expected atlantis config:\n%s\n",
			"Got atlantis config:\n%s",
		}, "\n"), prettyPrint(expectedConfig), prettyPrint(completeConfig))
	}
}

func TestFileExists(t *testing.T) {
	absPath := prepEnv(t)
	if !fileExists(absPath) {
		t.Errorf("File expected to exist, but didn't: %q", absPath)
	}
	if fileExists(absPath + "potato") {
		t.Errorf("File expected not to exist, but did: %q", absPath)
	}
}

func TestMakeProjectConfig(t *testing.T) {
	absPath := prepEnv(t)

	dependencies := map[string][]string{
		absPath + "/project1":        {"../modules/module1"},
		absPath + "/modules/module1": {"../module2"},
	}

	gotConfig := makeProjectConfig(absPath+"/project1", dependencies)

	expectedConfig := ProjectConfig{
		Name: "project1",
		Dir:  "project1",
		Autoplan: AutoplanConfig{
			Enabled: true,
			WhenModified: []string{
				"**/*",
				"../modules/module1/**/*",
				"../modules/module2/**/*",
			},
		},
	}

	if !reflect.DeepEqual(expectedConfig, gotConfig) {
		t.Errorf(strings.Join([]string{
			"Expected project config:\n%s\n",
			"Got project config:\n%s",
		}, "\n"), prettyPrint(expectedConfig), prettyPrint(gotConfig))
	}
}

func TestGetWhenModifiedPaths(t *testing.T) {
	absPath := prepEnv(t)

	dependencies := map[string][]string{
		absPath + "/project1":        {"../modules/module1"},
		absPath + "/project2":        {"../modules/module2"},
		absPath + "/modules/module1": {"../module2"},
		absPath + "/modules/module2": {},
	}

	gotPaths := getWhenModifiedPaths(absPath+"/project1", dependencies)

	expectedPaths := []string{
		absPath + "/project1/../modules/module1/**/*",
		absPath + "/project1/../modules/module1/../module2/**/*",
	}

	if !reflect.DeepEqual(expectedPaths, gotPaths) {
		t.Errorf(strings.Join([]string{
			"Expected paths:\n%s\n",
			"Got paths:\n%s",
		}, "\n"), prettyPrint(expectedPaths), prettyPrint(gotPaths))
	}
}

func TestCleanPaths(t *testing.T) {
	absPath := prepEnv(t)

	dirtyPaths := []string{
		absPath + "/project1/../modules/module1/**/*",
		absPath + "/project1/../modules/module1/../module2/**/*",
	}

	expectedPaths := []string{
		"**/*",
		"../modules/module1/**/*",
		"../modules/module2/**/*",
	}

	gotPaths := cleanPaths(dirtyPaths, absPath+"/project1")

	if !reflect.DeepEqual(expectedPaths, gotPaths) {
		t.Errorf(strings.Join([]string{
			"Expected paths:\n%s\n",
			"Got paths:\n%s",
		}, "\n"), prettyPrint(expectedPaths), prettyPrint(gotPaths))
	}
}

func TestUnique(t *testing.T) {
	dupes := []string{"thing 1", "thing 2", "thing 2"}

	expected := []string{"thing 1", "thing 2"}

	got := unique(dupes)

	if !reflect.DeepEqual(expected, got) {
		t.Errorf(strings.Join([]string{
			"Expected paths:\n%s\n",
			"Got paths:\n%s",
		}, "\n"), prettyPrint(expected), prettyPrint(got))
	}
}

func TestWriteAtlantisYaml(t *testing.T) {
	absPath := prepEnv(t)

	atlantisConfig := AtlantisConfig{
		Automerge:                 true,
		DeleteSourceBranchOnMerge: true,
		ParallelApply:             true,
		ParallelPlan:              true,
		Version:                   4,
	}

	expectedYaml := strings.Join([]string{
		"automerge: true",
		"delete_source_branch_on_merge: true",
		"parallel_apply: true",
		"parallel_plan: true",
		"projects: []",
		"version: 4\n",
	}, "\n")

	writeAtlantisYaml(atlantisConfig)
	gotYaml, err := ioutil.ReadFile(absPath + "/atlantis.yaml")

	if err != nil {
		t.Error(err)
	}

	revertAtlantisYaml(absPath)

	if expectedYaml != string(gotYaml) {
		t.Errorf(strings.Join([]string{
			"Expected yaml:\n%s\n",
			"Got yaml:\n%s\n",
		}, "\n"), expectedYaml, gotYaml)
	}
}

func TestMain(t *testing.T) {
	absPath := prepEnv(t)

	expectedYaml := strings.Join([]string{
		"automerge: true",
		"delete_source_branch_on_merge: true",
		"parallel_apply: true",
		"parallel_plan: true",
		"projects:",
		"    - autoplan:",
		"        enabled: true",
		"        when_modified:",
		"            - '**/*'",
		"            - ../modules/module1/**/*",
		"            - ../modules/module2/**/*",
		"      name: project1",
		"      dir: project1",
		"    - autoplan:",
		"        enabled: true",
		"        when_modified:",
		"            - '**/*'",
		"            - ../modules/module2/**/*",
		"      name: project2",
		"      dir: project2",
		"version: 3\n",
	}, "\n")

	main()

	gotYaml, err := ioutil.ReadFile(absPath + "/atlantis.yaml")

	if err != nil {
		t.Error(err)
	}

	revertAtlantisYaml(absPath)

	if expectedYaml != string(gotYaml) {
		t.Errorf(strings.Join([]string{
			"Expected yaml:\n%s\n",
			"Got yaml:\n%s\n",
		}, "\n"), expectedYaml, gotYaml)
	}
}
