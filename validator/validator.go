package validator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// ValidateByExt validates content based on file extension.
func ValidateByExt(filename string, content string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".json":
		return validateJSON(content)
	case ".yaml", ".yml":
		return validateYAML(content)
	case ".toml":
		return validateTOML(content)
	default:
		// If it looks like .env, validate basic KEY=VAL lines; otherwise accept.
		if looksLikeDotEnv(content) {
			return validateDotEnv(content)
		}
		return nil
	}
}

func validateJSON(content string) error {
	var v any
	dec := json.NewDecoder(strings.NewReader(content))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return fmt.Errorf("JSON parse error: %w", err)
	}
	return nil
}

func validateYAML(content string) error {
	var v any
	if err := yaml.Unmarshal([]byte(content), &v); err != nil {
		return fmt.Errorf("YAML parse error: %w", err)
	}
	return nil
}

func validateTOML(content string) error {
	var v any
	if err := toml.Unmarshal([]byte(content), &v); err != nil {
		return fmt.Errorf("TOML parse error: %w", err)
	}
	return nil
}

func looksLikeDotEnv(s string) bool {
	sc := bufio.NewScanner(strings.NewReader(s))
	lines, matches := 0, 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines++
		if strings.Contains(line, "=") && !strings.HasPrefix(line, "=") {
			matches++
		}
	}
	return lines > 0 && matches > 0
}

func validateDotEnv(s string) error {
	sc := bufio.NewScanner(strings.NewReader(s))
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if !strings.Contains(t, "=") || strings.HasPrefix(t, "=") {
			return fmt.Errorf(".env parse error on line %d: expected KEY=VALUE", lineNo)
		}
		kv := strings.SplitN(t, "=", 2)
		key := strings.TrimSpace(kv[0])
		if key == "" || strings.ContainsAny(key, " \t\"'") {
			return fmt.Errorf(".env invalid key on line %d", lineNo)
		}
	}
	return nil
}
