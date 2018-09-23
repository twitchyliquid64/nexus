package buildvar

var (
	gitHash   string
	buildDate string
	isProd    string
)

// GitHash returns the git commit hash, if one was provided at build.
func GitHash() string {
	return gitHash
}

// BuildDate returns when the build was done, if one was provided at build.
func BuildDate() string {
	return buildDate
}

// IsProd returns true if the build was marked as a production build.
func IsProd() bool {
	return isProd != ""
}
