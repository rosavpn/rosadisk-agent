package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// #nosec G101 -- directory path, not a credential
const secretsDir = "/var/lib/rosadisk-agent"

func writeSecret(filename, data string) error {
	path := filepath.Join(secretsDir, filename)
	if err := os.MkdirAll(secretsDir, 0750); err != nil {
		return fmt.Errorf("failed to create secrets directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(data), 0400); err != nil {
		return fmt.Errorf("failed to write secret file %s: %w", filename, err)
	}
	return nil
}

func readSecret(filename string) (string, error) {
	// #nosec G304 -- path is constructed from trusted filename under fixed directory
	data, err := os.ReadFile(filepath.Join(secretsDir, filename))
	if err != nil {
		return "", fmt.Errorf("failed to read secret file %s: %w", filename, err)
	}
	return string(data), nil
}

func deleteSecret(filename string) error {
	path := filepath.Join(secretsDir, filename)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete secret file %s: %w", filename, err)
	}
	return nil
}

func hasSecret(filename string) bool {
	_, err := os.Stat(filepath.Join(secretsDir, filename))
	return err == nil
}

func WriteE2EEKey(passphrase string) error {
	return writeSecret("e2ee_key", passphrase)
}

func ReadE2EEKey() (string, error) {
	return readSecret("e2ee_key")
}

func WriteS3AccessKey(key string) error {
	return writeSecret("aws_access_key", key)
}

func WriteS3SecretKey(key string) error {
	return writeSecret("aws_secret_key", key)
}

func HasE2EEKey() bool {
	return hasSecret("e2ee_key")
}

func ClearSecrets() error {
	if err := deleteSecret("e2ee_key"); err != nil {
		return err
	}
	if err := deleteSecret("aws_access_key"); err != nil {
		return err
	}
	if err := deleteSecret("aws_secret_key"); err != nil {
		return err
	}
	return nil
}
