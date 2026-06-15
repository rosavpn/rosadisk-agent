package config

type GlobalConfig struct {
	Backup   JobSchedule `json:"backup"`
	Snapshot JobSchedule `json:"snapshot"`
	Defrag   JobSchedule `json:"defrag"`
	Scrub    JobSchedule `json:"scrub"`
	Balance  JobSchedule `json:"balance"`
}

type JobSchedule struct {
	Time    string `json:"time"`
	Enabled bool   `json:"enabled"`
}

func DefaultConfig() GlobalConfig {
	return GlobalConfig{
		Backup:   JobSchedule{Enabled: true, Time: "04:00"},
		Snapshot: JobSchedule{Enabled: true, Time: "03:00"},
		Defrag:   JobSchedule{Enabled: true, Time: "04:00"},
		Scrub:    JobSchedule{Enabled: true, Time: "05:00"},
		Balance:  JobSchedule{Enabled: true, Time: "01:00"},
	}
}
