package database

import (
	"database/sql"
	"fmt"
	"time"
)

type JobLogRecord struct {
	ID          int64
	JobType     string
	Mountpoint  string
	SubvolumeID string
	TargetName  string
	Status      string
	Output      string
	Error       string
	StartedAt   time.Time
	CompletedAt time.Time
}

func InsertJobLog(db *sql.DB, r JobLogRecord) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO job_logs (job_type, mountpoint, subvolume_id, target_name, status, output, error, started_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, r.JobType, nullString(r.Mountpoint), nullString(r.SubvolumeID), nullString(r.TargetName),
		r.Status, nullString(r.Output), nullString(r.Error),
		r.StartedAt, r.CompletedAt)
	if err != nil {
		return 0, fmt.Errorf("failed to insert job log: %w", err)
	}

	return result.LastInsertId()
}

func UpdateJobLog(db *sql.DB, id int64, status, output, errMsg string) error {
	_, err := db.Exec(`
		UPDATE job_logs SET status = ?, output = ?, error = ?, completed_at = ? WHERE id = ?
	`, status, nullString(output), nullString(errMsg), time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update job log: %w", err)
	}

	return nil
}

func ListJobLogs(db *sql.DB, jobType, status string, limit int) ([]JobLogRecord, error) {
	query := `
		SELECT id, job_type, mountpoint, subvolume_id, target_name, status, output, error, started_at, completed_at
		FROM job_logs
		WHERE 1=1`
	args := make([]interface{}, 0)

	if jobType != "" {
		query += " AND job_type = ?"
		args = append(args, jobType)
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	query += " ORDER BY id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query job logs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []JobLogRecord
	for rows.Next() {
		var r JobLogRecord
		var mountpoint, subvolumeID, targetName, output, errMsg sql.NullString
		var completedAt sql.NullTime

		err := rows.Scan(&r.ID, &r.JobType, &mountpoint, &subvolumeID, &targetName,
			&r.Status, &output, &errMsg, &r.StartedAt, &completedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job log: %w", err)
		}

		r.Mountpoint = mountpoint.String
		r.SubvolumeID = subvolumeID.String
		r.TargetName = targetName.String
		r.Output = output.String
		r.Error = errMsg.String
		if completedAt.Valid {
			r.CompletedAt = completedAt.Time
		}

		records = append(records, r)
	}

	return records, nil
}

func GetJobLog(db *sql.DB, id int64) (*JobLogRecord, error) {
	var r JobLogRecord
	var mountpoint, subvolumeID, targetName, output, errMsg sql.NullString
	var completedAt sql.NullTime

	err := db.QueryRow(`
		SELECT id, job_type, mountpoint, subvolume_id, target_name, status, output, error, started_at, completed_at
		FROM job_logs
		WHERE id = ?
	`, id).Scan(&r.ID, &r.JobType, &mountpoint, &subvolumeID, &targetName,
		&r.Status, &output, &errMsg, &r.StartedAt, &completedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job log not found")
		}
		return nil, fmt.Errorf("failed to query job log: %w", err)
	}

	r.Mountpoint = mountpoint.String
	r.SubvolumeID = subvolumeID.String
	r.TargetName = targetName.String
	r.Output = output.String
	r.Error = errMsg.String
	if completedAt.Valid {
		r.CompletedAt = completedAt.Time
	}

	return &r, nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
