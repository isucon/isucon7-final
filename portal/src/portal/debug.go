package portal

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"expvar"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

var (
	AppVersion   = "undefined"
	AppStartedAt = time.Now()
)

func ServeDebugExpvar(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, "{\n")

	fmt.Fprintf(w, "%q: ", "db")
	json.NewEncoder(w).Encode(db.Stats())

	fmt.Fprintf(w, ",\n%q: ", "runtime")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"NumGoroutine": runtime.NumGoroutine(),
	})

	rows, err := db.Query("SELECT status,COUNT(*) FROM queues GROUP BY status")
	if err != nil {
		return err
	}
	defer rows.Close()
	queueStats := map[string]int{}
	for rows.Next() {
		var (
			st string
			c  int
		)
		rows.Scan(&st, &c)
		queueStats[st] = c
	}

	fmt.Fprintf(w, ",\n%q: ", "queue")
	json.NewEncoder(w).Encode(queueStats)

	fmt.Fprintf(w, ",\n%q: ", "app")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"version":   AppVersion,
		"startedAt": AppStartedAt,
	})

	fmt.Fprintf(w, ",\n%q: ", "benchmarker")
	benchmarkNodesMtx.Lock()
	for _, node := range benchmarkNodes {
		json.NewEncoder(w).Encode(node)
	}
	benchmarkNodesMtx.Unlock()

	expvar.Do(func(kv expvar.KeyValue) {
		fmt.Fprintf(w, ",\n")
		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(w, "\n}\n")

	return nil
}

func ServeDebugQueue(w http.ResponseWriter, req *http.Request) error {
	rows, err := db.Query(`
      SELECT
        queues.id,team_id,queues.group,name,status,queues.ip_address,IFNULL(bench_node, ''),LEFT(IFNULL(result_json, ''), 300),LEFT(IFNULL(log_text, ''), 100),created_at
      FROM queues
        LEFT JOIN teams ON queues.team_id = teams.id
      ORDER BY queues.id DESC
      LIMIT 100
	`)
	if err != nil {
		return err
	}

	type queueItem struct {
		ID        int
		TeamID    int
		TeamName  string
		Group     string
		Status    string
		IPAddr    string
		BenchNode string
		Result    string
		Log       string
		Time      time.Time
	}

	type viewParamsDebugQueue struct {
		viewParamsLayout
		Items []*queueItem
	}

	items := []*queueItem{}

	defer rows.Close()
	for rows.Next() {
		var item queueItem
		err := rows.Scan(&item.ID, &item.TeamID, &item.Group, &item.TeamName, &item.Status, &item.IPAddr, &item.BenchNode, &item.Result, &item.Log, &item.Time)
		if err != nil {
			return err
		}

		items = append(items, &item)
	}

	return templates["debug-queue.tmpl"].Execute(w,
		viewParamsDebugQueue{
			viewParamsLayout: viewParamsLayout{nil, contestDayNumber},
			Items:            items,
		})
}

func ServeDebugResult(w http.ResponseWriter, req *http.Request) error {
	id := req.URL.Query().Get("id")
	if len(id) == 0 {
		return errHTTP(http.StatusBadRequest)
	}

	var res string
	err := db.QueryRow(`SELECT IFNULL(result_json, '') FROM queues WHERE id = ?`, id).Scan(&res)
	if err == sql.ErrNoRows {
		return errHTTP(http.StatusNotFound)
	}
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	err = json.Indent(buf, []byte(res), "", "  ")
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	buf.WriteTo(w)

	return nil
}

func ServeDebugLog(w http.ResponseWriter, req *http.Request) error {
	id := req.URL.Query().Get("id")
	if len(id) == 0 {
		return errHTTP(http.StatusBadRequest)
	}

	var res string
	err := db.QueryRow(`SELECT IFNULL(log_text, '') FROM queues WHERE id = ?`, id).Scan(&res)
	if err == sql.ErrNoRows {
		return errHTTP(http.StatusNotFound)
	}
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, res)

	return nil
}

func ServeDebugQueueJob(w http.ResponseWriter, req *http.Request) error {
	servers := []string{
		"app2011.isu7f.k0y.org,app2012.isu7f.k0y.org,app2013.isu7f.k0y.org", // go
		"app2021.isu7f.k0y.org,app2022.isu7f.k0y.org,app2023.isu7f.k0y.org",
		"app2031.isu7f.k0y.org,app2032.isu7f.k0y.org,app2033.isu7f.k0y.org",
		"app2041.isu7f.k0y.org,app2042.isu7f.k0y.org,app2043.isu7f.k0y.org",
		"app2051.isu7f.k0y.org,app2052.isu7f.k0y.org,app2053.isu7f.k0y.org",
		"app2061.isu7f.k0y.org,app2062.isu7f.k0y.org,app2063.isu7f.k0y.org",
	}
	groups := []string{
		"sac-tk1a-sv221",
		"sac-tk1a-sv221",
		"sac-tk1a-sv223",
		"sac-tk1a-sv223",
		"sac-tk1a-sv225",
		"sac-tk1a-sv225",
	}

	insertIDs := []int{}
	for idx, sv := range servers {
		res, err := db.Exec("INSERT INTO queues (team_id, `group`, ip_address) VALUES (?, ?, ?)", 9999, groups[idx], sv)
		if err != nil {
			return err
		}

		k, err := res.LastInsertId()
		if err != nil {
			return err
		}
		insertIDs = append(insertIDs, int(k))
	}

	for _, id := range insertIDs {
		fmt.Fprintf(w, "%d\n", id)
	}

	return nil
}

// 全チームのベンチを実行する
func ServeDebugQueueAllTeam(w http.ResponseWriter, req *http.Request) error {
	type team struct {
		ID      int
		Name    string
		IPAddrs string
		Group   string

		JobID int
	}

	teams := []*team{}
	rows, err := db.Query("SELECT id, `group`, name, IFNULL(ip_address, '') FROM teams WHERE id <> 9999 ORDER BY id ASC")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		t := team{}
		err := rows.Scan(&t.ID, &t.Group, &t.Name, &t.IPAddrs)
		if err != nil {
			return err
		}
		teams = append(teams, &t)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, t := range teams {
		res, err := db.Exec("INSERT INTO queues (team_id, `group`, ip_address) VALUES (?, ?, ?)", 9999, t.Group, t.IPAddrs)
		if err != nil {
			return err
		}

		k, err := res.LastInsertId()
		if err != nil {
			return err
		}
		t.JobID = int(k)
	}

	for _, t := range teams {
		fmt.Fprintf(w, "%d\t%d\t%s\t%s\n", t.ID, t.JobID, t.IPAddrs, t.Name)
	}

	return nil
}
