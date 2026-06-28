package config

import (
	"encoding/json"
	"fmt"

	"rosadisk-agent/internal/database"
)

func InitConfig(db *database.Database) error {
	_, err := db.Exec(`REPLACE INTO global_config (id, data) VALUES (1, ?)`,
		jsonString(DefaultConfig()),
	)
	if err != nil {
		return fmt.Errorf("failed to init global config: %w", err)
	}
	return nil
}

func GetConfig(db *database.Database) (GlobalConfig, error) {
	var raw string
	err := db.QueryRow(`SELECT data FROM global_config WHERE id = 1`).Scan(&raw)
	if err != nil {
		return GlobalConfig{}, fmt.Errorf("failed to read global config: %w", err)
	}

	var cfg GlobalConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return GlobalConfig{}, fmt.Errorf("failed to parse global config: %w", err)
	}

	cfg.Encryption.Active = HasE2EEKey()

	return cfg, nil
}

func SaveConfig(db *database.Database, cfg GlobalConfig) error {
	cfg.Encryption.Active = false
	delete(cfg.BackupStorage.Options, "aws_access_key")
	delete(cfg.BackupStorage.Options, "aws_secret_key")

	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal global config: %w", err)
	}

	_, err = db.Exec(`UPDATE global_config SET data = ? WHERE id = 1`, string(data))
	if err != nil {
		return fmt.Errorf("failed to save global config: %w", err)
	}

	return nil
}

func jsonString(cfg GlobalConfig) string {
	data, _ := json.Marshal(cfg)
	return string(data)
}
