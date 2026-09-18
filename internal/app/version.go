package app

// applicationVersion is reported in the about dialog.
//
// The release workflow writes the version it releases into this line, with
// .github/scripts/set-version.sh, and commits it. Keep the line as it is:
// the script finds it by its shape, which a test in this package checks.
const applicationVersion = "0.1.0-dev"
