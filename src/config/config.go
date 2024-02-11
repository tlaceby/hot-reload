package config

const CONFIG_FILE_NAME = "config.hotreload.json"

type Config struct {
	WatchFileTypes []string // [ts, js, html, css] [*] means all
	IncludePaths   []string // [./src, main/foo]   [.] means cwd
	ExcludePaths   []string
	Commands       []string // [echo "Hello world"]
	Delay          int      // Time in ms
}

func CreateDefaultConfig() Config {
	return Config{
		WatchFileTypes: []string{"*"},
		IncludePaths:   []string{"."},
		ExcludePaths:   []string{},
		Commands:       []string{`echo "Files Modified: .MODIFIED_NAMES"`},
		Delay:          100,
	}
}
