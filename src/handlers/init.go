package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"

	"github.com/tlaceby/hot-reload/src/config"
	"github.com/tlaceby/hot-reload/src/errors"
)

func InitHandler(args []string) {
	cwd, _ := os.Getwd()
	useForce := slices.Contains(args, "-force")
	configFilePath := filepath.Join(cwd, config.CONFIG_FILE_NAME)
	info, err := os.Stat(configFilePath)

	exists := bool(err == nil)

	if !exists || (exists && useForce) {
		createConfig(configFilePath)
		return
	}

	if info.IsDir() {
		errors.Error("The config path should not be a directory.")
	}

	errors.Error("The config file already exists. Use the -force flag to override the file.")

}

func createConfig(path string) {
	fh, err := os.Create(path)
	errors.CheckErr(err)
	defer fh.Close()

	defaultConfig := config.CreateDefaultConfig()
	data, _ := json.MarshalIndent(defaultConfig, "", "	")

	_, err = fh.Write(data)
	errors.CheckErr(err)
}
