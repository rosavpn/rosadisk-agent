package server

import (
	"time"

	"rosadisk-agent/api/gen"
	"rosadisk-agent/internal/config"
	"rosadisk-agent/internal/database"
)

func toJobLogSummary(r database.JobLogRecord) gen.JobLogSummary {
	return gen.JobLogSummary{
		CompletedAt: nilTime(r.CompletedAt),
		Id:          int(r.ID),
		JobType:     r.JobType,
		Mountpoint:  nullStringPtr(r.Mountpoint),
		StartedAt:   r.StartedAt,
		Status:      r.Status,
		SubvolumeId: nullStringPtr(r.SubvolumeID),
		TargetName:  nullStringPtr(r.TargetName),
	}
}

func nilTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func nullStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toVolumeJobSchedule(v gen.VolumeJobSchedule) config.VolumeJobSchedule {
	c := config.VolumeJobSchedule{
		Enabled: v.Enabled,
		Time:    v.Time,
	}
	if v.HourlyMinute != nil {
		c.HourlyMinute = *v.HourlyMinute
	}
	if v.WeeklyDay != nil {
		c.WeeklyDay = string(*v.WeeklyDay)
	}
	if v.MonthlyDay != nil {
		c.MonthlyDay = *v.MonthlyDay
	}
	return c
}

func toDiskJobSchedule(d gen.DiskJobSchedule) config.DiskJobSchedule {
	c := config.DiskJobSchedule{
		Enabled:   d.Enabled,
		Frequency: string(d.Frequency),
		Time:      d.Time,
	}
	if d.DayOfWeek != nil {
		c.DayOfWeek = string(*d.DayOfWeek)
	}
	if d.DayOfMonth != nil {
		c.DayOfMonth = *d.DayOfMonth
	}
	return c
}
