package common

// Version information - set by main.go during build
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
	BuiltBy = "unknown"
)

// SetVersionInfo sets the version information from main.go
func SetVersionInfo(version, commit, date, builtBy string) {
	Version = version
	Commit = commit
	Date = date
	BuiltBy = builtBy
}

// GetVersion returns the current version
func GetVersion() string {
	return Version
}
