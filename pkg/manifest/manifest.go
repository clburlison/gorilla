package manifest

import (
	"io/ioutil"
	"path/filepath"

	"github.com/1dustindavis/gorilla/pkg/config"
	"github.com/1dustindavis/gorilla/pkg/download"
	"github.com/1dustindavis/gorilla/pkg/gorillalog"
	"gopkg.in/yaml.v2"
)

// Item represents a single object from the manifest
type Item struct {
	Name       string   `yaml:"name"`
	Includes   []string `yaml:"included_manifests"`
	Installs   []string `yaml:"managed_installs"`
	Uninstalls []string `yaml:"managed_uninstalls"`
	Updates    []string `yaml:"managed_updates"`
	Catalogs   []string `yaml:"catalogs"`
}

func getManifest(manifestName string) Item {
	// Unmarshal the yaml file
	yamlPath := filepath.Join(config.CachePath, manifestName) + ".yaml"
	yamlFile, err := ioutil.ReadFile(yamlPath)
	if err != nil {
		gorillalog.Error("Unable to read manifest:", yamlFile, err)
	}
	var manifest Item
	err = yaml.Unmarshal(yamlFile, &manifest)
	if err != nil {
		gorillalog.Error("Unable to parse yaml manifest:", yamlPath, err)
	}
	return manifest
}

// This abstraction allows us to override when testing
var downloadFile = download.File

// Get returns a slice the includes all manifest objects
func Get() []Item {
	// Create a slice of all manifest objects
	var manifests []Item
	// Create a slice with the names of all manifests
	// This is so we can track them before we get the data
	var manifestsList []string

	// Setup iteration tracking for manifests
	var manifestsTotal int
	var manifestsProcessed = 0
	var manifestsRemaining = 1

	// Add the top level manifest to the list
	manifestsList = append(manifestsList, config.Current.Manifest)

	for manifestsRemaining > 0 {
		currentManifest := manifestsList[manifestsProcessed]

		// Add the current manifest to our working list
		workingList := []string{currentManifest}

		// Download the manifest
		manifestURL := config.Current.URL + "manifests/" + currentManifest + ".yaml"
		gorillalog.Info("Manifest Url:", manifestURL)
		err := downloadFile(config.CachePath, manifestURL)
		if err != nil {
			gorillalog.Error("Unable to retrieve manifest:", currentManifest, err)
		}

		// Get new manifest
		newManifest := getManifest(currentManifest)

		// Add any includes to our working list
		workingList = append(workingList, newManifest.Includes...)

		// Get workingList unique items, and add to the real list
		for _, item := range workingList {

			// Check if unique in manifestsList
			var uniqueInList = true
			for i := range manifestsList {
				if manifestsList[i] == item {
					uniqueInList = false
				}
			}
			// Update manifestsList if it is unique
			if uniqueInList {
				manifestsList = append(manifestsList, item)
			}
		}

		// Check if this is unique in manifests
		var uniqueInManifests = true
		for i := range manifests {
			if manifests[i].Name == newManifest.Name {
				uniqueInManifests = false
			}
		}
		// Update manifests
		if uniqueInManifests {
			// manifests = append([]Item{newManifest}, manifests...)
			manifests = append(manifests, newManifest)
		}

		// If any catalogs are in the manifest, append them to the end of the list
		for _, newCatalog := range newManifest.Catalogs {
			// Before adding it, check if it is already on the list
			var match bool
			for _, oldCatalog := range config.Current.Catalogs {
				if oldCatalog == newCatalog {
					match = true
				}
			}
			// If "match" is still false, it is not already on the list
			if !match {
				config.Current.Catalogs = append(config.Current.Catalogs, newCatalog)
			}
		}

		// Increment counters
		manifestsTotal = len(manifestsList)
		manifestsProcessed++
		manifestsRemaining = manifestsTotal - manifestsProcessed
	}
	return manifests
}
