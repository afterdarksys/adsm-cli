// Package config loads and persists adsm's on-disk configuration.
//
// Layout (~/.adsm/config.yaml):
//
//	current_profile: default
//	profiles:
//	  default:
//	    api_url: https://api.afterdarksys.com
//	    account: ryan@afterdarksys.com
//
// Credentials are deliberately NOT stored here — see the note on Config.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultAPIURL is the public API endpoint used when nothing overrides it.
const DefaultAPIURL = "https://api.afterdarksys.com"

// DefaultProfile is the profile used when --profile is not given.
const DefaultProfile = "default"

// Config is the resolved configuration for a single profile.
//
// It intentionally holds no tokens or secrets. Credentials belong in the OS
// keychain via the credentials store (internal/auth), not in a world-readable
// YAML file. Adding a Token field here would be a regression, not a shortcut.
type Config struct {
	// Profile is the name this config was loaded under.
	Profile string `yaml:"-"`

	// APIURL is the API endpoint for this profile.
	APIURL string `yaml:"api_url"`

	// Account is the identity last authenticated on this profile, stored for
	// display only ("logged in as ..."). It is not proof of authentication.
	Account string `yaml:"account,omitempty"`

	// OrganizationID is the explicitly selected canonical tenant. It is an
	// identifier, never an email-derived tenancy key.
	OrganizationID string `yaml:"organization_id,omitempty"`

	// Output is the default output format when -o is not supplied.
	Output string `yaml:"output,omitempty"`
}

// file is the on-disk shape of config.yaml.
type file struct {
	CurrentProfile string             `yaml:"current_profile"`
	Profiles       map[string]*Config `yaml:"profiles"`
}

// Dir returns the adsm configuration directory, honouring ADSM_CONFIG_DIR.
func Dir() (string, error) {
	if d := os.Getenv("ADSM_CONFIG_DIR"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".adsm"), nil
}

// Path returns the full path to config.yaml.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Load reads the named profile. A missing config file is not an error: it
// returns defaults, so 'adsm login' works on a clean machine.
func Load(profile string) (*Config, error) {
	if profile == "" {
		profile = DefaultProfile
	}

	path, err := Path()
	if err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{Profile: profile, APIURL: DefaultAPIURL, Output: "table"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var f file
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	c, ok := f.Profiles[profile]
	if !ok || c == nil {
		return nil, fmt.Errorf("profile %q not found in %s", profile, path)
	}

	c.Profile = profile
	if c.APIURL == "" {
		c.APIURL = DefaultAPIURL
	}
	if c.Output == "" {
		c.Output = "table"
	}
	return c, nil
}

// Save writes this profile back to config.yaml, preserving other profiles.
//
// TODO(scaffold): not yet wired to any command. 'adsm login' will be the
// first caller.
func Save(c *Config) error {
	if c == nil {
		return errors.New("nil config")
	}
	profile := c.Profile
	if profile == "" {
		profile = DefaultProfile
	}

	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}

	path := filepath.Join(dir, "config.yaml")

	f := file{Profiles: map[string]*Config{}}
	if raw, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(raw, &f); err != nil {
			return fmt.Errorf("parsing existing %s: %w", path, err)
		}
		if f.Profiles == nil {
			f.Profiles = map[string]*Config{}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	f.Profiles[profile] = c
	if f.CurrentProfile == "" {
		f.CurrentProfile = profile
	}

	out, err := yaml.Marshal(&f)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	// 0600: the file carries account identity and endpoints, and should not be
	// group- or world-readable even though it holds no tokens.
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
