package portal

import "time"

type Job struct {
	ID      int    `json:"id"`
	TeamID  int    `json:"team_id"`
	IPAddrs string `json:"ip_addrs"`
}

type BenchResult struct {
	JobID   string `json:"job_id"`
	IPAddrs string `json:"ip_addrs"`

	Pass      bool     `json:"pass"`
	Score     int64    `json:"score"`
	Message   string   `json:"message"`
	Errors    []string `json:"error"`
	Logs      []string `json:"log"`
	LoadLevel int      `json:"load_level"`

	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

type Result struct {
	Job   *Job
	Bench *BenchResult
	At    time.Time
}
