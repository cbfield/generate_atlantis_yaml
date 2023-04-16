package main

import (
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/hashicorp/terraform/configs"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

// https://www.runatlantis.io/docs/custom-workflows.html#step
// "DIR - Absolute path to the current directory."
//
// This represents the root of the repository during Atlantis pre-workflow hook execution.
var ROOT = os.Getenv("DIR")

var IGNORE_DIRS = []string{".circleci", ".git", ".github", ".terraform"}

type AtlantisConfig struct {
	Automerge                 bool            `yaml:"automerge"`
	DeleteSourceBranchOnMerge bool            `yaml:"delete_source_branch_on_merge"`
	ParallelApply             bool            `yaml:"parallel_apply"`
	ParallelPlan              bool            `yaml:"parallel_plan"`
	Projects                  []ProjectConfig `yaml:"projects"`
	Version                   int             `yaml:"version"`
}

type ProjectConfig struct {
	Autoplan AutoplanConfig `yaml:"autoplan"`
	Name     string         `yaml:"name"`
	Dir      string         `yaml:"dir"`
}

type AutoplanConfig struct {
	Enabled      bool     `yaml:"enabled"`
	WhenModified []string `yaml:"when_modified"`
}

// Get a list of projects and a map of path dependencies for each project.
//
// Walk the directory tree, starting at the root of the repository. For each directory:
//   If it is on the ignore list, skip it and its children.
//   If it contains Terraform configuration files, load their contents into a module struct.
//   If the module has backend config, add the directory of the module to the projects list.
//   If the module calls other modules, add their sources to the dependencies of the directory.
func getProjectsAndDependencies() ([]string, map[string][]string) {
	projects := []string{}
	dependencies := make(map[string][]string)

	err := filepath.Walk(ROOT, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		if slices.Contains(IGNORE_DIRS, info.Name()) {
			return filepath.SkipDir
		}

		dependencies[path] = []string{}

		if tfconfig.IsModuleDir(path) {
			parser := configs.NewParser(nil)
			module, diags := parser.LoadConfigDir(path)

			if diags.HasErrors() {
				log.Fatal(diags)
			}

			if module.Backend != nil {
				projects = append(projects, path)
			}

			for _, moduleCall := range module.ModuleCalls {
				absPath := filepath.Join(path, moduleCall.SourceAddr)

				if fileExists(absPath) && !slices.Contains(dependencies[path], moduleCall.SourceAddr) {
					dependencies[path] = append(dependencies[path], moduleCall.SourceAddr)
				}
			}
		}

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	return projects, dependencies
}

// Read the contents of `atlantis.yaml` and reflect them into an AtlantisConfig struct.
// `atlantis.yaml` is expected to exist at the path "${ROOT}/atlantis.yaml"
func readAtlantisYaml() AtlantisConfig {
	atlantis_yaml := ROOT + "/atlantis.yaml"

	file, err := ioutil.ReadFile(atlantis_yaml)

	if err != nil {
		log.Fatal(err)
	}

	var atlantisConfig AtlantisConfig
	err = yaml.Unmarshal(file, &atlantisConfig)

	if err != nil {
		log.Fatal(err)
	}

	return atlantisConfig
}

// Add project configurations to the atlantis config.
// This is done with goroutines because its easy and they make it go zoom zoom real fast.
// Explanation here: https://gobyexample.com/waitgroups
func addProjectsToConfig(atlantisConfig AtlantisConfig, projects []string, dependencies map[string][]string) AtlantisConfig {
	// If `projects` configurations exist already, overwrite them instead of appending to them.
	atlantisConfig.Projects = []ProjectConfig{}

	wg := sync.WaitGroup{}
	wg.Add(len(projects))

	for i := 0; i < len(projects); i++ {
		go func(i int) {
			projectConfig := makeProjectConfig(projects[i], dependencies)
			atlantisConfig.Projects = append(atlantisConfig.Projects, projectConfig)
			defer wg.Done()
		}(i)
	}

	wg.Wait()
	return atlantisConfig
}

// Check if there exists a file located at the given path.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

// Make the project configurations for a single project.
func makeProjectConfig(project string, dependencies map[string][]string) ProjectConfig {
	whenModifiedPaths := getWhenModifiedPaths(project, dependencies)
	cleanedPaths := cleanPaths(whenModifiedPaths, project)
	projectRelativePath := strings.Replace(project, ROOT+"/", "", 1)

	projectConfig := ProjectConfig{
		Autoplan: AutoplanConfig{
			Enabled:      true,
			WhenModified: cleanedPaths,
		},
		Dir:  projectRelativePath,
		Name: projectRelativePath,
	}

	return projectConfig
}

// For a given project, list the relative paths from that project's directory
// to the directories containing modules that the project depends on.
//
// This is done recursively. When a module directory is identified as a path dependency,
// we also check for dependencies of that module, and so on, since changes to those
// submodules may affect the resources managed by the project.
//
// The paths returned by this function are kinda gross (e.g. "abs/path/to/project1/../modules/module1/../module2")
// This is because we can't clean the paths while the function recurses.
// The cleaning is done after the full list is generated.
func getWhenModifiedPaths(path string, dependencies map[string][]string) []string {
	paths := []string{}

	// If we are recursing, `path` represents a potentially messy
	// absolute path to a module that our project depends on.
	// (e.g. "abs/path/to/project1/../modules/module1/../module2")
	//
	// The `dependencies` map is keyed with relative paths from the root of the repository
	// (e.g. `modules/module2`), so we reformat `path` to match that format.
	cleanPath := filepath.Clean(path)

	for _, dep := range dependencies[cleanPath] {
		paths = append(paths, path+"/"+dep+"/**/*")
		paths = append(paths, getWhenModifiedPaths(path+"/"+dep, dependencies)...)
	}

	return paths
}

// Take the paths generated for a project by getWhenModifiedPaths,
// - ("abs/path/to/project1/../modules/module1/../module2")
//
// and turn them into relative paths from the project directory with wildcards.
// - ("../modules/module2/**/*")
//
// We add wildcards because we want to autoplan based on changes to any files
// in any subdirectories of each module, in addition to the root directory.
func cleanPaths(paths []string, project string) []string {
	cleanedPaths := []string{}

	for _, path := range paths {
		cleanedPath := strings.Replace(path, project+"/", "", 1)
		cleanedPath = filepath.Clean(cleanedPath)
		cleanedPaths = append(cleanedPaths, cleanedPath)
	}

	cleanedPaths = append(cleanedPaths, "**/*")
	cleanedPaths = unique(cleanedPaths)
	sort.Strings(cleanedPaths)

	return cleanedPaths
}

// Take a list of file paths and return the same list without duplicates.
func unique(paths []string) []string {
	allKeys := make(map[string]bool)
	uniquePaths := []string{}

	for _, item := range paths {
			if _, value := allKeys[item]; !value {
					allKeys[item] = true
					uniquePaths = append(uniquePaths, item)
			}
	}

	return uniquePaths
}

// Take an AtlantisConfig struct, encode it into yaml, and write it to `atlantis.yaml`.
func writeAtlantisYaml(atlantisConfig AtlantisConfig) {
	atlantisYaml, err := yaml.Marshal(&atlantisConfig)

	if err != nil {
		log.Fatal(err)
	}

	absPath := ROOT + "/atlantis.yaml"
	err = ioutil.WriteFile(absPath, atlantisYaml, 0)

	if err != nil {
		log.Fatal(err)
	}
}

// Walk the repository and gather a list of project directories
// and a map of their dependencies (also directories).
//
// Load content of `atlantis.yaml` into a struct.
// Add autoplan configurations for each project.
//
// Encode contents into yaml and write it back to the file.
func main() {
	projects, dependencies := getProjectsAndDependencies()

	atlantisConfig := readAtlantisYaml()
	atlantisConfigComplete := addProjectsToConfig(atlantisConfig, projects, dependencies)

	writeAtlantisYaml(atlantisConfigComplete)
}
