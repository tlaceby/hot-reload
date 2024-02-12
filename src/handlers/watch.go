package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/tlaceby/hot-reload/src/config"
	"github.com/tlaceby/hot-reload/src/errors"
)

func WatchHandler(args []string) {
	cwd, _ := os.Getwd()
	configFilePath := filepath.Join(cwd, config.CONFIG_FILE_NAME)
	info, err := os.Stat(configFilePath)

	if !(err != nil || !info.IsDir()) {
		errors.Error(fmt.Sprintf("Missing expected config file: %s", config.CONFIG_FILE_NAME))
	}

	data, err := ioutil.ReadFile(configFilePath)
	errors.HandleErr(err, fmt.Sprintf("Could not read the config file: %s", configFilePath))

	config := config.Config{}
	err = json.Unmarshal(data, &config)
	errors.HandleErr(err, "Could not load/parse json config")

	// Specify Default Values
	config.Delay = int(math.Max(float64(config.Delay), 10))

	if len(config.IncludePaths) == 0 {
		config.IncludePaths = append(config.IncludePaths, ".")
	}

	if len(config.WatchFileTypes) == 0 {
		config.WatchFileTypes = append(config.WatchFileTypes, "*")
	}

	if len(config.Commands) == 0 {
		config.Commands = append(config.Commands, `echo "File(s) Changed: .MODIFIED_NAMES"`)
	}

	var wg sync.WaitGroup

	for _, watchPath := range config.IncludePaths {
		var path string

		if len(watchPath) == 0 {
			continue
		}

		isAbsolutePath := watchPath[0] == '/'

		if isAbsolutePath {
			path = watchPath
		} else {
			path = filepath.Join(cwd, watchPath)
		}

		wg.Add(1)
		go func() {
			watchFilePath(path, config)
			defer wg.Done()
		}()
	}

	wg.Wait()
}

type ModifiedStatus struct {
	Path   string
	Status string
}

func watchFilePath(watchPath string, config config.Config) {
	pathLastUpdatedLU := map[string]time.Time{}

	for {
		modified := []ModifiedStatus{}
		pathsChecked := []string{watchPath}
		pathInfo, err := os.Stat(watchPath)

		if err != nil {
			time.Sleep(time.Millisecond * time.Duration(config.Delay))
			continue
		}

		modifiedForWatchPath, exists := pathLastUpdatedLU[watchPath]
		updatedModtime := pathInfo.ModTime()

		if !exists || modifiedForWatchPath.Before(updatedModtime) {
			status := "Modified"

			if !exists {
				status = "Created"
			}

			pathLastUpdatedLU[watchPath] = updatedModtime
			modified = append(modified, ModifiedStatus{
				Path: watchPath, Status: status,
			})
		}

		// If the parent folder is changes/renamed/created/deleted etc... then no need to recurse.
		if len(modified) == 0 {
			for _, path := range listDirContents(watchPath, config.WatchFileTypes) {
				previousModified, exists := pathLastUpdatedLU[path]
				info, err := os.Stat(path)
				pathsChecked = append(pathsChecked, path)

				if err != nil {
					// File/folder deleted right inbetween checks
					if exists {
						modified = append(modified, ModifiedStatus{
							Path:   path,
							Status: "Deleted",
						})

						delete(pathLastUpdatedLU, path)
					}

					continue
				}

				if !exists || previousModified.Before(info.ModTime()) {
					status := "Modified"

					if !exists {
						status = "Created"
					}

					modified = append(modified, ModifiedStatus{
						Path:   path,
						Status: status,
					})

					pathLastUpdatedLU[path] = info.ModTime()
				}
			}

			for path := range pathLastUpdatedLU {
				if slices.Contains(pathsChecked, path) {
					continue
				}

				modified = append(modified, ModifiedStatus{
					Path:   path,
					Status: "Deleted",
				})

				delete(pathLastUpdatedLU, path)
			}
		}

		if len(modified) > 0 {
			fmt.Printf("Folder modified!\n -> %v\n", modified)
		}

		time.Sleep(time.Millisecond * time.Duration(config.Delay))
	}

}

// listDirContents recursively lists all files and directories within the given path.
func listDirContents(path string, allowedFileTypes []string) []string {
	var contents []string

	entries, err := os.ReadDir(path)
	if err != nil {
		return []string{path}
	}

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())

		if entry.IsDir() {
			contents = append(contents, listDirContents(fullPath, allowedFileTypes)...)
			continue
		}

		for _, extension := range allowedFileTypes {
			split := strings.Split(fullPath, ".")

			if len(split) > 1 && split[len(split)-1] == extension {
				contents = append(contents, fullPath)
			}
		}

	}

	return contents
}
