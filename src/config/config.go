package config

const CONFIG_FILE_NAME = "config.hotreload.json"

type Command struct {
	Command string
	Args    []string
}

type Config struct {
	WatchFileTypes []string // [ts, js, html, css] [*] means all
	IncludePaths   []string // [./src, main/foo]   [.] means cwd
	ExcludePaths   []string
	Commands       []Command // [echo "Hello world"]
	Delay          int       // Time in ms
}

func CreateDefaultConfig() Config {
	return Config{
		WatchFileTypes: []string{"*"},
		IncludePaths:   []string{"."},
		ExcludePaths:   []string{},
		Commands: []Command{
			{Command: "echo", Args: []string{"Files Changes: .MODIFIED"}},
		},
		Delay: 100,
	}
}
