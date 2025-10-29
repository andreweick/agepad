package model

import "testing"

func TestConfig(t *testing.T) {
	t.Run("creates valid config with all fields", func(t *testing.T) {
		cfg := Config{
			FilePath:       "/path/to/file.age",
			RecipientsFile: ".age-recipients",
			IdentitiesPath: "~/.config/age/key.txt",
			Armor:          true,
			ViewOnly:       false,
		}

		if cfg.FilePath != "/path/to/file.age" {
			t.Errorf("expected FilePath to be '/path/to/file.age', got %s", cfg.FilePath)
		}
		if cfg.RecipientsFile != ".age-recipients" {
			t.Errorf("expected RecipientsFile to be '.age-recipients', got %s", cfg.RecipientsFile)
		}
		if cfg.IdentitiesPath != "~/.config/age/key.txt" {
			t.Errorf("expected IdentitiesPath to be '~/.config/age/key.txt', got %s", cfg.IdentitiesPath)
		}
		if !cfg.Armor {
			t.Error("expected Armor to be true")
		}
		if cfg.ViewOnly {
			t.Error("expected ViewOnly to be false")
		}
	})
}

func TestRotateConfig(t *testing.T) {
	t.Run("creates valid rotate config with all fields", func(t *testing.T) {
		cfg := RotateConfig{
			Root:               ".",
			FromRecipientsFile: ".age-recipients",
			ToRecipientsFile:   ".age-recipients.new",
			IdentitiesPath:     "~/.config/age/key.txt",
		}

		if cfg.Root != "." {
			t.Errorf("expected Root to be '.', got %s", cfg.Root)
		}
		if cfg.FromRecipientsFile != ".age-recipients" {
			t.Errorf("expected FromRecipientsFile to be '.age-recipients', got %s", cfg.FromRecipientsFile)
		}
		if cfg.ToRecipientsFile != ".age-recipients.new" {
			t.Errorf("expected ToRecipientsFile to be '.age-recipients.new', got %s", cfg.ToRecipientsFile)
		}
		if cfg.IdentitiesPath != "~/.config/age/key.txt" {
			t.Errorf("expected IdentitiesPath to be '~/.config/age/key.txt', got %s", cfg.IdentitiesPath)
		}
	})
}

func TestRunConfig(t *testing.T) {
	t.Run("creates valid run config with all fields", func(t *testing.T) {
		cfg := RunConfig{
			FilePath:       "/path/to/secrets.env.age",
			IdentitiesPath: "~/.config/age/key.txt",
			Command:        []string{"myserver", "--port", "8080"},
		}

		if cfg.FilePath != "/path/to/secrets.env.age" {
			t.Errorf("expected FilePath to be '/path/to/secrets.env.age', got %s", cfg.FilePath)
		}
		if cfg.IdentitiesPath != "~/.config/age/key.txt" {
			t.Errorf("expected IdentitiesPath to be '~/.config/age/key.txt', got %s", cfg.IdentitiesPath)
		}
		if len(cfg.Command) != 3 {
			t.Errorf("expected Command to have 3 elements, got %d", len(cfg.Command))
		}
		if cfg.Command[0] != "myserver" {
			t.Errorf("expected Command[0] to be 'myserver', got %s", cfg.Command[0])
		}
	})
}
