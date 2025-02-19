package plugin_magefile

import "fmt"

const (
	ErrorsURL = "https://github.com/mattermost/mattermost-plugin-legal-hold/blob/main/plugin_magefile/ERROR_CODES.md#"

	ErrInitGnuTar = "init-gnu-tar"
	ErrInitGo     = "init-go"
	ErrInitNpm    = "init-npm"
)

// buildErrorURL builds the URL for the error code
func buildErrorURL(code string) string {
	return fmt.Sprintf("%s%s", ErrorsURL, code)
}

// buildErrorToDisplay builds the error message to display to the user
func buildErrorToDisplay(code string) string {
	return fmt.Sprintf("Error: %s (see %s)", code, buildErrorURL(code))
}
