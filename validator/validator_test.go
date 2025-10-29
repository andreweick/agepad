package validator

import (
	"strings"
	"testing"
)

func TestValidateJSON(t *testing.T) {
	t.Run("validates valid JSON", func(t *testing.T) {
		content := `{"key": "value", "number": 123}`
		err := ValidateByExt("test.json", content)
		if err != nil {
			t.Errorf("expected valid JSON to pass validation, got error: %v", err)
		}
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		content := `{"key": "value", invalid}`
		err := ValidateByExt("test.json", content)
		if err == nil {
			t.Error("expected invalid JSON to fail validation")
		}
		if !strings.Contains(err.Error(), "JSON parse error") {
			t.Errorf("expected JSON parse error, got: %v", err)
		}
	})

	t.Run("validates empty JSON object", func(t *testing.T) {
		content := `{}`
		err := ValidateByExt("test.json", content)
		if err != nil {
			t.Errorf("expected empty JSON object to pass validation, got error: %v", err)
		}
	})

	t.Run("validates JSON array", func(t *testing.T) {
		content := `[1, 2, 3, "test"]`
		err := ValidateByExt("test.json", content)
		if err != nil {
			t.Errorf("expected JSON array to pass validation, got error: %v", err)
		}
	})
}

func TestValidateYAML(t *testing.T) {
	t.Run("validates valid YAML", func(t *testing.T) {
		content := `
key: value
number: 123
list:
  - item1
  - item2
`
		err := ValidateByExt("test.yaml", content)
		if err != nil {
			t.Errorf("expected valid YAML to pass validation, got error: %v", err)
		}
	})

	t.Run("validates valid YML extension", func(t *testing.T) {
		content := `key: value`
		err := ValidateByExt("test.yml", content)
		if err != nil {
			t.Errorf("expected valid YML to pass validation, got error: %v", err)
		}
	})

	t.Run("rejects invalid YAML", func(t *testing.T) {
		content := `
key: value
  : invalid
`
		err := ValidateByExt("test.yaml", content)
		if err == nil {
			t.Error("expected invalid YAML to fail validation")
		}
		if err != nil && !strings.Contains(err.Error(), "YAML parse error") {
			t.Errorf("expected YAML parse error, got: %v", err)
		}
	})

	t.Run("validates empty YAML", func(t *testing.T) {
		content := ``
		err := ValidateByExt("test.yaml", content)
		if err != nil {
			t.Errorf("expected empty YAML to pass validation, got error: %v", err)
		}
	})
}

func TestValidateTOML(t *testing.T) {
	t.Run("validates valid TOML", func(t *testing.T) {
		content := `
title = "TOML Example"

[owner]
name = "Test User"
`
		err := ValidateByExt("test.toml", content)
		if err != nil {
			t.Errorf("expected valid TOML to pass validation, got error: %v", err)
		}
	})

	t.Run("rejects invalid TOML", func(t *testing.T) {
		content := `
title = "missing quote
invalid syntax
`
		err := ValidateByExt("test.toml", content)
		if err == nil {
			t.Error("expected invalid TOML to fail validation")
		}
		if !strings.Contains(err.Error(), "TOML parse error") {
			t.Errorf("expected TOML parse error, got: %v", err)
		}
	})

	t.Run("validates empty TOML", func(t *testing.T) {
		content := ``
		err := ValidateByExt("test.toml", content)
		if err != nil {
			t.Errorf("expected empty TOML to pass validation, got error: %v", err)
		}
	})
}

func TestValidateDotEnv(t *testing.T) {
	t.Run("validates valid .env file", func(t *testing.T) {
		content := `
# Comment
KEY1=value1
KEY2=value2
EMPTY_VALUE=
`
		err := ValidateByExt("test.env", content)
		if err != nil {
			t.Errorf("expected valid .env to pass validation, got error: %v", err)
		}
	})

	t.Run("validates .env with spaces in values", func(t *testing.T) {
		content := `KEY=value with spaces`
		err := ValidateByExt("test.env", content)
		if err != nil {
			t.Errorf("expected .env with spaces in value to pass validation, got error: %v", err)
		}
	})

	t.Run("rejects .env with invalid key", func(t *testing.T) {
		content := `INVALID KEY=value`
		err := ValidateByExt("test.env", content)
		if err == nil {
			t.Error("expected .env with space in key to fail validation")
		}
		if !strings.Contains(err.Error(), ".env invalid key") {
			t.Errorf("expected .env invalid key error, got: %v", err)
		}
	})

	t.Run("rejects .env with line starting with =", func(t *testing.T) {
		content := `KEY=value
=value`
		err := ValidateByExt("test.env", content)
		if err == nil {
			t.Error("expected .env with line starting with = to fail validation")
		}
		if err != nil && !strings.Contains(err.Error(), ".env parse error") {
			t.Errorf("expected .env parse error, got: %v", err)
		}
	})

	t.Run("rejects .env with line missing =", func(t *testing.T) {
		content := `KEY1=value
KEY_WITHOUT_EQUALS`
		err := ValidateByExt("test.env", content)
		if err == nil {
			t.Error("expected .env with missing = to fail validation")
		}
		if err != nil && !strings.Contains(err.Error(), ".env parse error") {
			t.Errorf("expected .env parse error, got: %v", err)
		}
	})

	t.Run("validates .env with only comments and blank lines", func(t *testing.T) {
		content := `
# Just comments

# More comments
`
		err := ValidateByExt("test.env", content)
		if err != nil {
			t.Errorf("expected .env with only comments to pass validation, got error: %v", err)
		}
	})
}

func TestValidateByExt(t *testing.T) {
	t.Run("accepts unknown file extension without validation", func(t *testing.T) {
		content := `This is just plain text`
		err := ValidateByExt("test.txt", content)
		if err != nil {
			t.Errorf("expected unknown extension to pass validation, got error: %v", err)
		}
	})

	t.Run("accepts file without extension", func(t *testing.T) {
		content := `Some content`
		err := ValidateByExt("testfile", content)
		if err != nil {
			t.Errorf("expected file without extension to pass validation, got error: %v", err)
		}
	})

	t.Run("detects .env format without .env extension", func(t *testing.T) {
		content := `KEY1=value1
KEY2=value2`
		err := ValidateByExt("config", content)
		if err != nil {
			t.Errorf("expected .env-like content to pass validation, got error: %v", err)
		}
	})
}

func TestLooksLikeDotEnv(t *testing.T) {
	t.Run("identifies content that looks like .env", func(t *testing.T) {
		content := `KEY=value`
		if !looksLikeDotEnv(content) {
			t.Error("expected content to be identified as .env-like")
		}
	})

	t.Run("rejects content without equals signs", func(t *testing.T) {
		content := `Just plain text`
		if looksLikeDotEnv(content) {
			t.Error("expected plain text to not be identified as .env-like")
		}
	})

	t.Run("rejects empty content", func(t *testing.T) {
		content := ``
		if looksLikeDotEnv(content) {
			t.Error("expected empty content to not be identified as .env-like")
		}
	})

	t.Run("rejects content with only comments", func(t *testing.T) {
		content := `# Just a comment`
		if looksLikeDotEnv(content) {
			t.Error("expected comment-only content to not be identified as .env-like")
		}
	})
}
