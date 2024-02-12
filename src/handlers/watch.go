package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
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

	configSettings := config.Config{}
	err = json.Unmarshal(data, &configSettings)
	errors.HandleErr(err, "Could not load/parse json config")

	// Specify Default Values
	configSettings.Delay = int(math.Max(float64(configSettings.Delay), 10))

	if len(configSettings.IncludePaths) == 0 {
		configSettings.IncludePaths = append(configSettings.IncludePaths, ".")
	}

	if len(configSettings.WatchFileTypes) == 0 {
		configSettings.WatchFileTypes = append(configSettings.WatchFileTypes, "*")
	}

	if len(configSettings.Commands) == 0 {
		configSettings.Commands = append(configSettings.Commands, config.Command{Command: "echo", Args: []string{"Files Changes: .MODIFIED"}})
	}

	var wg sync.WaitGroup

	for _, watchPath := range configSettings.IncludePaths {

		// Make sure all paths are absolute
		for i, ignorePath := range configSettings.ExcludePaths {
			if len(ignorePath) > 0 && ignorePath[0] != '/' {
				configSettings.ExcludePaths[i] = filepath.Join(cwd, ignorePath)
			}
		}

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
			watchFilePath(path, configSettings)
			defer wg.Done()
		}()
	}

	wg.Wait()
}

type ModifiedStatus struct {
	Path   string `json:"Path"`
	Status string `json:"Status"`
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
			for _, path := range listDirContents(watchPath, config.WatchFileTypes, config.ExcludePaths) {
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
			modifiedJson, _ := json.MarshalIndent(modified, "", " ")
			jsonStr := string(modifiedJson)

			for _, command := range config.Commands {
				var args []string = []string{}
				// Replace Templates
				for _, arg := range command.Args {
					if strings.Contains(arg, ".MODIFIED") {
						argStr := strings.ReplaceAll(arg, ".MODIFIED", jsonStr)
						args = append(args, argStr)
					}
				}

				output, err := exec.Command(command.Command, args...).Output()

				if err == nil {
					println(string(output))
				}
			}
		}

		time.Sleep(time.Millisecond * time.Duration(config.Delay))
	}

}

// listDirContents recursively lists all files and directories within the given path.
func listDirContents(path string, allowedFileTypes []string, ignorePaths []string) []string {
	allowAllTypes := slices.Contains(allowedFileTypes, "*")
	var contents []string

	entries, err := os.ReadDir(path)
	if err != nil {
		return []string{path}
	}

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())

		if slices.Contains(ignorePaths, fullPath) {
			continue
		}

		if entry.IsDir() {
			contents = append(contents, listDirContents(fullPath, allowedFileTypes, ignorePaths)...)
			continue
		}

		if allowAllTypes {
			contents = append(contents, fullPath)
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
