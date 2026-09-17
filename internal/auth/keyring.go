package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const keyringService = "afterdarksys.adsm"

type Credential struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	Expiry       time.Time `json:"expiry"`
}

type Store interface {
	Get(context.Context, string) (Credential, error)
	Put(context.Context, string, Credential) error
	Delete(context.Context, string) error
}

type commandRunner func(context.Context, string, []string, string) ([]byte, error)
type Keyring struct {
	GOOS string
	run  commandRunner
}

func NewKeyring() *Keyring {
	return &Keyring{GOOS: runtime.GOOS, run: func(ctx context.Context, name string, args []string, input string) ([]byte, error) {
		command := exec.CommandContext(ctx, name, args...)
		command.Stdin = strings.NewReader(input)
		return command.Output()
	}}
}

func (k *Keyring) Get(ctx context.Context, profile string) (Credential, error) {
	var out []byte
	var err error
	switch k.GOOS {
	case "darwin":
		out, err = k.run(ctx, "security", []string{"find-generic-password", "-s", keyringService, "-a", profile, "-w"}, "")
	case "linux":
		out, err = k.run(ctx, "secret-tool", []string{"lookup", "service", keyringService, "profile", profile}, "")
	default:
		return Credential{}, fmt.Errorf("OS keychain unsupported on %s", k.GOOS)
	}
	if err != nil {
		return Credential{}, errors.New("not logged in")
	}
	var value Credential
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(out))), &value); err != nil {
		return Credential{}, errors.New("invalid credential in OS keychain")
	}
	return value, nil
}
func (k *Keyring) Put(ctx context.Context, profile string, value Credential) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	switch k.GOOS {
	case "darwin":
		_, err = k.run(ctx, "security", []string{"add-generic-password", "-U", "-s", keyringService, "-a", profile, "-w", string(encoded)}, "")
	case "linux":
		_, err = k.run(ctx, "secret-tool", []string{"store", "--label", "After Dark Systems CLI", "service", keyringService, "profile", profile}, string(encoded))
	default:
		return fmt.Errorf("OS keychain unsupported on %s", k.GOOS)
	}
	return err
}
func (k *Keyring) Delete(ctx context.Context, profile string) error {
	var err error
	switch k.GOOS {
	case "darwin":
		_, err = k.run(ctx, "security", []string{"delete-generic-password", "-s", keyringService, "-a", profile}, "")
	case "linux":
		_, err = k.run(ctx, "secret-tool", []string{"clear", "service", keyringService, "profile", profile}, "")
	default:
		return fmt.Errorf("OS keychain unsupported on %s", k.GOOS)
	}
	return err
}
