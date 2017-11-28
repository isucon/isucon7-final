package portal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/pkg/errors"
)

type errAlreadyQueued int

func (n errAlreadyQueued) Error() string {
	return fmt.Sprintf("job already queued (teamID=%d)", n)
}

func enqueueJob(team *Team, ipAddr string) error {
	var id int
	err := db.QueryRow(`
      SELECT id FROM queues
      WHERE team_id = ? AND status IN ('waiting', 'running')`, team.ID).Scan(&id)
	switch {
	case err == sql.ErrNoRows:
		// 行がない場合はINSERTする
	case err != nil:
		return errors.Wrap(err, "failed to enqueue job when selecting table")
	default:
		return errAlreadyQueued(team.ID)
	}
	// XXX: worker nodeが死んだ時のために古くて実行中のジョブがある場合をケアした方が良いかも

	// XXX: ここですり抜けて二重で入る可能性がある
	_, err = db.Exec("INSERT INTO queues (team_id, `group`, ip_address) VALUES (?, ?, ?)", team.ID, team.Group, ipAddr)
	if err != nil {
		return errors.Wrap(err, "enqueue job failed")
	}
	return nil
}

func dequeueJob(benchNode BenchmarkNode) (*Job, error) {
	j := Job{}
	err := db.QueryRow("SELECT id, team_id, ip_address FROM queues WHERE status = 'waiting' AND `group`= ? ORDER BY id LIMIT 1", benchNode.Group).Scan(&j.ID, &j.TeamID, &j.IPAddrs)
	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, errors.Wrap(err, "dequeue job failed when scanning job")
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, errors.Wrap(err, "failed to dequeue job when beginning tx")
	}
	ret, err := tx.Exec(`
    UPDATE queues SET status = 'running', bench_node = ?
      WHERE id = ? AND status = 'waiting'`, benchNode.Name, j.ID)
	if err != nil {
		tx.Rollback()
		return nil, errors.Wrap(err, "failed to dequeue job when locking")
	}
	affected, err := ret.RowsAffected()
	if err != nil {
		tx.Rollback()
		return nil, errors.Wrap(err, "failed to dequeue job when checking affected rows")
	}
	if affected > 1 {
		tx.Rollback()
		return nil, fmt.Errorf("failed to dequeue job. invalid affected rows: %d", affected)
	}
	err = tx.Commit()
	if err != nil {
		return nil, errors.Wrap(err, "failed to dequeue job when commiting tx")
	}
	// タッチの差で別のワーカーにジョブを取られたとか
	if affected < 1 {
		return nil, nil
	}
	return &j, nil
}

func cancelJob(teamID int) error {
	j := Job{}
	err := db.QueryRow(`
    SELECT id, team_id, ip_address FROM queues
		  WHERE status = 'waiting' AND team_id = ? ORDER BY id LIMIT 1`, teamID).Scan(
		&j.ID, &j.TeamID, &j.IPAddrs)

	switch {
	case err == sql.ErrNoRows:
		return nil
	case err != nil:
		return errors.Wrap(err, "failed to cancel job when scanning job")
	}

	tx, err := db.Begin()
	if err != nil {
		return errors.Wrap(err, "failed to cancel job when beginning tx")
	}
	ret, err := tx.Exec(`
    UPDATE queues SET status = 'canceled'
      WHERE id = ? AND status = 'waiting'`, j.ID)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to cancel job when locking")
	}
	affected, err := ret.RowsAffected()
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to cancel job when checking affected rows")
	}
	if affected > 1 {
		tx.Rollback()
		return fmt.Errorf("failed to cancel job. invalid affected rows: %d", affected)
	}
	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to cancel job when commiting tx")
	}
	if affected < 1 {
		return nil
	}
	return nil
}

func doneJob(res *BenchResult, logText string) error {
	b, _ := json.Marshal(res)
	resultJSON := string(b)

	log.Printf("doneJob: result=%+v", res)

	tx, err := db.Begin()
	if err != nil {
		return errors.Wrap(err, "doneJob failed when beginning tx")
	}

	j := Job{}
	err = db.QueryRow(`SELECT id, team_id, ip_address FROM queues WHERE id = ?`, res.JobID).Scan(&j.ID, &j.TeamID, &j.IPAddrs)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "doneJob failed when find job")
	}

	ret, err := tx.Exec(`
    UPDATE queues SET status = 'done', result_json = ?, log_text = ? WHERE id = ? AND status = 'running'`,
		resultJSON, logText, res.JobID)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "doneJob failed when locking")
	}
	affected, err := ret.RowsAffected()
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "doneJob failed when checking affected rows")
	}
	if affected != 1 {
		tx.Rollback()
		return fmt.Errorf("doneJob failed. invalid affected rows=%d", affected)
	}

	if res.Pass {
		_, err = tx.Exec("INSERT INTO scores (team_id, score) VALUES (?, ?)", j.TeamID, res.Score)
		if err != nil {
			tx.Rollback()
			return errors.Wrap(err, "INSERT INTO scores")
		}
		_, err = tx.Exec(`
			INSERT INTO team_scores (team_id, latest_score, best_score)
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE
			best_score = GREATEST(best_score, VALUES(best_score)),
			latest_score = VALUES(latest_score)
		`,
			j.TeamID, res.Score, res.Score,
		)
		if err != nil {
			tx.Rollback()
			return errors.Wrap(err, "INSERT INTO team_scores")
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "doneJob failed when commiting tx")
	}
	return nil
}

func abortJob(jobID int, resultJson, logText string) error {
	_, err := db.Exec(`
    UPDATE queues SET status = 'aborted', result_json = ?, log_text = ? WHERE id = ? AND status = 'running'`,
		resultJson, logText, jobID)
	if err != nil {
		return errors.Wrap(err, "abortJob failed when exec")
	}
	return nil
}

func checkTimeoutJob() (int, error) {
	result := `{"reason":"CheckTimeoutJob"}`
	ret, err := db.Exec(`
		UPDATE queues SET status = 'aborted', result_json = ?
		WHERE status = 'running' AND updated_at < (NOW() - INTERVAL 150 SECOND)`, result)
	if err != nil {
		return 0, errors.Wrap(err, "checkTimeoutJob failed when update")
	}

	affected, err := ret.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "checkTimeoutJob failed when checking affected rows")
	}

	return int(affected), nil
}
