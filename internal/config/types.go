package config

type GlobalConfig struct {
	Backup        VolumeJobSchedule   `json:"backup"`
	Snapshot      VolumeJobSchedule   `json:"snapshot"`
	Defrag        VolumeJobSchedule   `json:"defrag"`
	Scrub         DiskJobSchedule     `json:"scrub"`
	Balance       DiskJobSchedule     `json:"balance"`
	BackupStorage BackupStorageConfig `json:"backup_storage"`
	Encryption    EncryptionConfig    `json:"encryption"`
}

type BackupStorageConfig struct {
	Type    string            `json:"type"`
	Options map[string]string `json:"options"`
}

type EncryptionConfig struct {
	Active bool `json:"active"`
}

type VolumeJobSchedule struct {
	Enabled      bool   `json:"enabled"`
	Time         string `json:"time"`
	HourlyMinute int    `json:"hourly_minute"`
	WeeklyDay    string `json:"weekly_day"`
	MonthlyDay   int    `json:"monthly_day"`
}

type DiskJobSchedule struct {
	Enabled    bool   `json:"enabled"`
	Frequency  string `json:"frequency"`
	Time       string `json:"time"`
	DayOfWeek  string `json:"day_of_week"`
	DayOfMonth int    `json:"day_of_month"`
}

func DefaultConfig() GlobalConfig {
	return GlobalConfig{
		Backup: VolumeJobSchedule{
			Enabled:      true,
			Time:         "03:00",
			HourlyMinute: 0,
			WeeklyDay:    "monday",
			MonthlyDay:   1,
		},
		Snapshot: VolumeJobSchedule{
			Enabled:      true,
			Time:         "03:00",
			HourlyMinute: 0,
			WeeklyDay:    "monday",
			MonthlyDay:   1,
		},
		Defrag: VolumeJobSchedule{
			Enabled:      true,
			Time:         "04:00",
			HourlyMinute: 0,
			WeeklyDay:    "monday",
			MonthlyDay:   1,
		},
		Scrub: DiskJobSchedule{
			Enabled:    true,
			Frequency:  "monthly",
			Time:       "05:00",
			DayOfWeek:  "sunday",
			DayOfMonth: 1,
		},
		Balance: DiskJobSchedule{
			Enabled:    true,
			Frequency:  "monthly",
			Time:       "05:00",
			DayOfWeek:  "sunday",
			DayOfMonth: 1,
		},
		BackupStorage: BackupStorageConfig{
			Type:    "",
			Options: map[string]string{},
		},
		Encryption: EncryptionConfig{
			Active: false,
		},
	}
}
