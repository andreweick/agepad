package model

// Config holds the configuration for the TUI editor mode.
type Config struct {
	FilePath       string
	RecipientsFile string
	IdentitiesPath string
	Armor          bool
	ViewOnly       bool
}

// RotateConfig holds the configuration for the rotate subcommand.
type RotateConfig struct {
	Root             string
	FromRecipientsFile string
	ToRecipientsFile string
	IdentitiesPath   string
}

// RunConfig holds the configuration for the run subcommand.
type RunConfig struct {
	FilePath       string
	IdentitiesPath string
	Command        []string
}
