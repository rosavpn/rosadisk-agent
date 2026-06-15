package config

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

func InitConfig(db *sql.DB) error {
	_, err := db.Exec(`INSERT OR IGNORE INTO global_config (id, data) VALUES (1, ?)`,
		jsonString(DefaultConfig()),
	)
	if err != nil {
		return fmt.Errorf("failed to init global config: %w", err)
	}
	return nil
}

func GetConfig(db *sql.DB) (GlobalConfig, error) {
	var raw string
	err := db.QueryRow(`SELECT data FROM global_config WHERE id = 1`).Scan(&raw)
	if err != nil {
		return GlobalConfig{}, fmt.Errorf("failed to read global config: %w", err)
	}

	var cfg GlobalConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return GlobalConfig{}, fmt.Errorf("failed to parse global config: %w", err)
	}

	return cfg, nil
}

func SaveConfig(db *sql.DB, cfg GlobalConfig) error {
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
