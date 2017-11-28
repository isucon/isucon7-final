package portal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	"golang.org/x/sync/singleflight"
)

var (
	templates    = map[string]*template.Template{}
	sessionStore sessions.Store

	benchmarkNodesMtx sync.Mutex
	benchmarkNodes    = map[string]BenchmarkNode{}
)

const (
	sessionName      = "isu7f-portal"
	sessionKeyTeamID = "team-id"

	rankingPickLatest = 20
)

func InitWeb() error {
	const templatesRoot = "views/"

	for _, file := range []string{
		"index.tmpl",
		"admin.tmpl",
		"login.tmpl",
		"admin-server.tmpl",
		"debug-queue.tmpl",
		"debug-leaderboard.tmpl",
	} {
		t := template.New(file).Funcs(template.FuncMap{
			"contestEnded": func() bool {
				return GetContestStatus() == ContestStatusEnded
			},
			"since": func(t time.Time) string {
				return fmt.Sprint(time.Since(t))
			},
			"noescape": func(s string) template.HTML {
				return template.HTML(s)
			},
			"joinslice": func(v []string) string {
				return strings.Join(v, "\n")
			},
		})

		if err := parseTemplateAsset(t, templatesRoot+"layout.tmpl"); err != nil {
			return err
		}

		if err := parseTemplateAsset(t, templatesRoot+file); err != nil {
			return err
		}

		templates[file] = t
	}

	// 日によってDBを分けるので、万一 teams.id が被ってたら
	// 前日のセッションでログイン状態になってしまう
	sessionStore = sessions.NewCookieStore([]byte(fmt.Sprintf(":sushi::beers:%d", contestDayNumber)))

	return nil
}

type BenchmarkNode struct {
	Name       string
	State      string
	Group      string
	IPAddr     string
	LastAccess time.Time
}

func parseTemplateAsset(t *template.Template, name string) error {
	content, err := Asset(name)
	if err != nil {
		return err
	}

	_, err = t.Parse(string(content))
	return err
}

type Team struct {
	ID           int
	Name         string
	Group        string
	IPAddr       string
	InstanceName string
	Category     string
}

func (t *Team) IsAdmin() bool {
	return t.ID == 9999
}

type Score struct {
	Team   Team
	Latest int64
	Best   int64
	At     time.Time
}

func loadTeam(id uint64) (*Team, error) {
	var team Team

	row := db.QueryRow("SELECT id,name,`group`,IFNULL(ip_address, ''),IFNULL(instance_name, '') FROM teams WHERE id = ?", id)
	err := row.Scan(&team.ID, &team.Name, &team.Group, &team.IPAddr, &team.InstanceName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &team, err
}

func loadTeamFromSession(req *http.Request) (*Team, error) {
	if *debugMode {
		c, _ := req.Cookie("debug_team")
		if c != nil {
			n, _ := strconv.ParseUint(c.Value, 10, 0)
			if n != 0 {
				return loadTeam(n)
			}
		}
	}

	sess, err := sessionStore.New(req, sessionName)
	if err != nil {
		if cerr, ok := err.(securecookie.Error); ok && cerr.IsDecode() {
			// 違う session secret でアクセスしにくるとこれなので無視
		} else {
			return nil, errors.Wrap(err, "sessionStore.New()")
		}
	}

	v, ok := sess.Values[sessionKeyTeamID]
	if !ok {
		return nil, nil
	}

	teamID, ok := v.(uint64)
	if !ok {
		return nil, nil
	}

	team, err := loadTeam(teamID)
	return team, errors.Wrapf(err, "loadTeam(id=%#v)", teamID)
}

type queuedJob struct {
	TeamID int
	Status string
}

type viewParamsLayout struct {
	Team *Team
	Day  int
}

type teamServer struct {
	ID       string
	Name     string
	LocalIP  string
	GlobalIP string
	Selected bool
}

type PlotLine struct {
	TeamName string           `json:"name"`
	Data     map[string]int64 `json:"data"`
}

type viewParamsIndex struct {
	viewParamsLayout
	Ranking        []*Score
	RankingIsFixed bool
	Jobs           []queuedJob
	TeamServers    []*teamServer
	LatestResult   *Result
	Score          *Score
	Message        string
	Info           string
	PlotLines      []*PlotLine
}

type viewParamsLogin struct {
	viewParamsLayout
	ErrorMessage string
}

func ServeIndex(w http.ResponseWriter, req *http.Request) error {
	message := ""

	queryMessage := req.URL.Query().Get("message")
	if queryMessage == "job_already_queued" {
		message = "Job already queued"
	}

	return ServeIndexWithMessage(w, req, message)
}

func buildLeaderboard(team *Team) ([]*Score, *Score, bool, error) {
	// team_scores_snapshot にデータが入ってたらそっちを使う
	// ラスト1時間でランキングの更新を止めるための措置
	// データは手動でいれる :P
	ranking, myScore, err := buildLeaderboardFromTable(team, true)
	if err == nil && ranking != nil && len(ranking) > 0 {
		return ranking, myScore, true, nil
	} else if err != nil {
		log.Printf("buildLeaderboardFromTable: %v", err)
	}

	ranking, myScore, err = buildLeaderboardFromTable(team, false)
	return ranking, myScore, false, nil
}

func buildScorePlot(team *Team) ([]*PlotLine, error) {
	// まずはsnapshotを見に行って、そこにデータがあればそれを使う
	plotLines, err := buildScorePlotFromTable(team, true)
	if err == nil && plotLines != nil && len(plotLines) > 0 {
		return plotLines, nil
	} else if err != nil {
		log.Printf("buildLeaderboardFromTable: %v", err)
	}

	// なかったらsnapshotじゃないものを見る
	plotLines, err = buildScorePlotFromTable(team, false)
	return plotLines, err
}

var plotGroup singleflight.Group

func buildScorePlotFromTable(team *Team, useSnapshot bool) ([]*PlotLine, error) {
	table := "scores"
	if useSnapshot {
		table = "scores_snapshot"
	}

	result, err, _ := plotGroup.Do(table, func() (interface{}, error) {
		rows, err := db.Query(`
		SELECT teams.id,teams.name,scores.score,scores.created_at
		FROM ` + table + ` AS scores
		  JOIN teams
		  ON scores.team_id = teams.id
		WHERE teams.id <> 9999
	`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		plotLines := make(map[int]*PlotLine)

		for rows.Next() {
			var (
				teamName string
				teamID   int
				score    int64
				at       time.Time
			)

			err := rows.Scan(&teamID, &teamName, &score, &at)
			if err != nil {
				return nil, err
			}

			if plotLines[teamID] == nil {
				plotLines[teamID] = &PlotLine{
					TeamName: teamName,
					Data:     map[string]int64{},
				}
			}

			plotLines[teamID].Data[at.Format("2006-01-02T15:04:05")] = score
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return plotLines, nil
	})

	if err != nil {
		return nil, err
	}

	plotLines := result.(map[int]*PlotLine)

	teamPlotLine := &PlotLine{
		TeamName: team.Name,
		Data:     map[string]int64{},
	}

	// snapshot の場合でも自分のスコアだけは最新のものにする(運営の場合を除く)
	if useSnapshot && !team.IsAdmin() {
		rows, err := db.Query(`SELECT score, created_at FROM scores WHERE team_id = ?`, team.ID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var (
				score int64
				at    time.Time
			)

			err := rows.Scan(&score, &at)
			if err != nil {
				return nil, err
			}

			teamPlotLine.Data[at.Format("2006-01-02T15:04:05")] = score
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	} else {
		if v, ok := plotLines[team.ID]; ok {
			teamPlotLine = v
		}
	}

	resultPlotLines := []*PlotLine{teamPlotLine}
	for teamID, plotLine := range plotLines {
		if teamID != team.ID {
			resultPlotLines = append(resultPlotLines, plotLine)
		}
	}

	return resultPlotLines, nil
}

func buildLeaderboardFromTable(team *Team, useSnapshot bool) ([]*Score, *Score, error) {
	// ランキングを作る。
	// 現在のスコアのトップ rankingPickLatest と自チーム
	table := "team_scores"
	if useSnapshot {
		table = "team_scores_snapshot"
	}

	var (
		allScores     = []*Score{}
		scoreByTeamID = map[int]*Score{}
	)

	rows, err := db.Query(`
		SELECT teams.id,teams.name,team_scores.latest_score,team_scores.best_score,team_scores.updated_at,teams.category
		FROM ` + table + ` AS team_scores
		  JOIN teams
		  ON team_scores.team_id = teams.id
		WHERE teams.category <> 'official'
	`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var score Score
		err := rows.Scan(&score.Team.ID, &score.Team.Name, &score.Latest, &score.Best, &score.At, &score.Team.Category)
		if err != nil {
			return nil, nil, err
		}
		allScores = append(allScores, &score)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	if useSnapshot {
		if len(allScores) == 0 {
			// スナップショットテーブルが空のときはさっさと処理を終わる
			return []*Score{}, nil, nil
		}

		// スナップショットの場合も自分のスコアだけは最新を使う
		if team.ID == 9999 {
			// 運営チームがランキングに入ってたら混乱するので入れない
		} else {
			var score Score
			err := db.QueryRow(`
				SELECT teams.id,teams.name,team_scores.latest_score,team_scores.best_score,team_scores.updated_at,teams.category
				FROM team_scores
				  JOIN teams
				  ON team_scores.team_id = teams.id
				WHERE teams.id = ?
			`, team.ID).Scan(&score.Team.ID, &score.Team.Name, &score.Latest, &score.Best, &score.At, &score.Team.Category)
			if err != nil {
				return nil, nil, err
			}

			scoreByTeamID[score.Team.ID] = &score
		}
	} else {
		for _, score := range allScores {
			if score.Team.ID == team.ID {
				scoreByTeamID[score.Team.ID] = score
			}
		}
	}

	sort.Slice(allScores, func(i, j int) bool {
		return allScores[i].Latest > allScores[j].Latest
	})

	for i, s := range allScores {
		if i >= rankingPickLatest {
			break
		}
		if s.Team.ID != team.ID {
			scoreByTeamID[s.Team.ID] = s
		}
	}

	ranking := make([]*Score, 0, len(scoreByTeamID))
	for _, s := range scoreByTeamID {
		ranking = append(ranking, s)
	}

	// 最後に、最新のスコアでソート
	sort.Slice(ranking, func(i, j int) bool {
		return ranking[i].Latest > ranking[j].Latest
	})

	return ranking, scoreByTeamID[team.ID], nil
}

func ServeIndexWithMessage(w http.ResponseWriter, req *http.Request, message string) error {
	team, err := loadTeamFromSession(req)
	if err != nil {
		return err
	}

	if team == nil {
		http.Redirect(w, req, "/login", http.StatusFound)
		return nil
	}

	if !team.IsAdmin() {
		if GetContestStatus() == ContestStatusNotStarted {
			http.Error(w, "Today's contest has not started yet", http.StatusForbidden)
			return nil
		}
	}

	ranking, myScore, rankingIsFixed, err := buildLeaderboard(team)
	if err != nil {
		return err
	}

	plotLines, err := buildScorePlot(team)
	if err != nil {
		return err
	}

	// キューをゲット
	jobs := []queuedJob{}
	if GetContestStatus() == ContestStatusStarted {
		rows, err := db.Query(`
			SELECT team_id, status
			FROM queues
			WHERE status IN ('waiting', 'running')
			  AND team_id <> 9999
			ORDER BY id ASC
		`)
		if err != nil {
			return err
		}
		for rows.Next() {
			var job queuedJob
			err := rows.Scan(&job.TeamID, &job.Status)
			if err != nil {
				rows.Close()
				return err
			}
			jobs = append(jobs, job)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
	}

	// 自分チームの最新状況を取得
	var (
		status       string
		latestResult *Result
		latestAt     time.Time
		latestJson   string
	)
	err = db.QueryRow(`
		SELECT status,IFNULL(result_json, ''),updated_at FROM queues
		WHERE team_id = ?
		  AND status IN ('done', 'canceled', 'aborted')
		ORDER BY updated_at DESC
		LIMIT 1
	`, team.ID).Scan(&status, &latestJson, &latestAt)
	if err == sql.ErrNoRows {
		err = nil
	}
	if err != nil {
		return err
	}

	if status == "done" {
		var br BenchResult
		err := json.Unmarshal([]byte(latestJson), &br)
		if err != nil {
			return err
		}
		latestResult = &Result{
			Bench: &br,
			At:    latestAt,
		}
	} else if status == "canceled" {
		latestResult = &Result{
			Bench: &BenchResult{
				Message: "ベンチマークがキャンセルされました。このスコアは反映されません。",
				Score:   0,
			},
			At: latestAt,
		}
	} else if status == "aborted" {
		latestResult = &Result{
			Bench: &BenchResult{
				Message: "ベンチマークが中断されました。このスコアは反映されません。",
				Score:   0,
			},
			At: latestAt,
		}
	}

	servers := []*teamServer{}
	rows, err := db.Query(`
		SELECT id, name, local_ip, global_ip FROM servers
		WHERE team_id = ?
		ORDER BY name ASC
	`, team.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var sv teamServer
		err := rows.Scan(&sv.ID, &sv.Name, &sv.LocalIP, &sv.GlobalIP)
		if err != nil {
			rows.Close()
			return err
		}
		sv.ID = strings.TrimSpace(sv.ID)
		sv.Name = strings.TrimSpace(sv.Name)
		sv.LocalIP = strings.TrimSpace(sv.LocalIP)
		sv.GlobalIP = strings.TrimSpace(sv.GlobalIP)
		servers = append(servers, &sv)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, addr := range strings.Split(team.IPAddr, ",") {
		for _, sv := range servers {
			if addr == sv.Name {
				sv.Selected = true
			}
		}
	}

	return templates["index.tmpl"].Execute(
		w, viewParamsIndex{
			viewParamsLayout: viewParamsLayout{
				Team: team,
				Day:  contestDayNumber,
			},
			Info:           infoText,
			Ranking:        ranking,
			RankingIsFixed: rankingIsFixed,
			TeamServers:    servers,
			Jobs:           jobs,
			Message:        message,
			LatestResult:   latestResult,
			Score:          myScore,
			PlotLines:      plotLines,
		},
	)
}

func ServeLogin(w http.ResponseWriter, req *http.Request) error {
	team, err := loadTeamFromSession(req)
	if err != nil {
		return err
	}

	if req.Method == "GET" {
		return templates["login.tmpl"].Execute(w, viewParamsLogin{viewParamsLayout{team, contestDayNumber}, ""})
	}

	var (
		id       = req.FormValue("team_id")
		password = req.FormValue("password")
	)

	var teamID uint64
	row := db.QueryRow("SELECT id FROM teams WHERE id = ? AND password = ? LIMIT 1", id, password)
	err = row.Scan(&teamID)
	if err != nil {
		if err == sql.ErrNoRows {
			return templates["login.tmpl"].Execute(w, viewParamsLogin{viewParamsLayout{team, contestDayNumber}, "Wrong id/password pair"})
		} else {
			return err
		}
	}

	sess, err := sessionStore.New(req, sessionName)
	if err != nil {
		if cerr, ok := err.(securecookie.Error); ok && cerr.IsDecode() {
			// 違う session secret でアクセスしにくるとこれなので無視
		} else {
			return errors.Wrap(err, "sessionStore.New()")
		}
	}

	sess.Values[sessionKeyTeamID] = teamID

	err = sess.Save(req, w)
	if err != nil {
		return err
	}

	http.Redirect(w, req, "/", 302)

	return nil
}

type httpError interface {
	httpStatus() int
	error
}

type errHTTP int

func (s errHTTP) Error() string   { return http.StatusText(int(s)) }
func (s errHTTP) HttpStatus() int { return int(s) }

type errHTTPMessage struct {
	status  int
	message string
}

func (m errHTTPMessage) Error() string   { return m.message }
func (m errHTTPMessage) HttpStatus() int { return m.status }

func ServeStatic(w http.ResponseWriter, req *http.Request) error {
	path := req.URL.Path[1:]
	content, err := Asset(path)
	if err != nil {
		return errHTTP(http.StatusNotFound)
	}
	if strings.HasSuffix(path, ".css") {
		w.Header().Set("Content-Type", "text/css")
	}
	w.Write(content)

	return nil
}

func ServeUpdateTeam(w http.ResponseWriter, req *http.Request) error {
	if GetContestStatus() == ContestStatusEnded {
		http.Error(w, "Today's contest has ended", http.StatusForbidden)
		return nil
	}

	if req.Method != http.MethodPost {
		return errHTTP(http.StatusMethodNotAllowed)
	}

	err := req.ParseForm()
	if err != nil {
		return err
	}

	team, err := loadTeamFromSession(req)
	if err != nil {
		return err
	}
	if team == nil {
		return errHTTP(http.StatusForbidden)
	}

	servers := []*teamServer{}
	rows, err := db.Query(`
		SELECT id, name, local_ip, global_ip FROM servers
		WHERE team_id = ?
		ORDER BY id ASC
	`, team.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var sv teamServer
		err := rows.Scan(&sv.ID, &sv.Name, &sv.LocalIP, &sv.GlobalIP)
		if err != nil {
			rows.Close()
			return err
		}
		servers = append(servers, &sv)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	selected := req.Form["servers[]"]
	selectedIPs := []string{}
	for _, sv := range servers {
		for _, sel := range selected {
			if strings.TrimSpace(sel) == sv.ID {
				selectedIPs = append(selectedIPs, sv.Name)
				break
			}
		}
	}

	teamIPAddr := strings.Join(selectedIPs, ",")

	err = updateTeamIPAddr(team, teamIPAddr)
	if err != nil {
		return errors.Wrapf(err, "updateTeamIPAddr(team=%#v, ipAddr=%#v)", team, teamIPAddr)
	}

	http.Redirect(w, req, "/", http.StatusFound)
	return nil
}

func ServeDebugLeaderboard(w http.ResponseWriter, req *http.Request) error {
	// ここは常に最新のを使う
	ranking, _, err := buildLeaderboardFromTable(&Team{}, false)
	if err != nil {
		return err
	}

	plotLines, err := buildScorePlotFromTable(&Team{}, false)
	if err != nil {
		return err
	}

	type viewParamsDebugLeaderboard struct {
		viewParamsLayout
		Ranking   []*Score
		PlotLines []*PlotLine
	}

	return templates["debug-leaderboard.tmpl"].Execute(
		w, viewParamsDebugLeaderboard{
			viewParamsLayout{nil, contestDayNumber},
			ranking,
			plotLines,
		},
	)
}

// serveQueueJob は参加者がベンチマーカのジョブをキューに挿入するエンドポイント。
func ServeQueueJob(w http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodPost {
		return errHTTP(http.StatusMethodNotAllowed)
	}

	// コンテスト開始前、終了後はエンキューさせない
	switch GetContestStatus() {
	case ContestStatusNotStarted:
		return errHTTPMessage{http.StatusForbidden, "Contest has not started yet"}
	case ContestStatusEnded:
		return errHTTPMessage{http.StatusForbidden, "Contest has finished"}
	}

	team, err := loadTeamFromSession(req)
	if err != nil {
		return err
	}
	if team == nil {
		return errHTTP(http.StatusForbidden)
	}

	err = req.ParseForm()
	if err != nil {
		return err
	}

	if req.Form.Get("cancel") != "" {
		err = cancelJob(team.ID)
		if err != nil {
			return err
		}

		http.Redirect(w, req, "/", http.StatusFound)
		return nil
	}

	ipAddr := team.IPAddr // IP設定しないと Enqueue 押せないはず
	if team.IsAdmin() {
		ipAddr = strings.TrimSpace(req.Form.Get("ipaddr"))
	}

	if ipAddr == "" {
		return errHTTP(http.StatusBadRequest)
	}

	err = enqueueJob(team, ipAddr)
	if err != nil {
		if _, ok := err.(errAlreadyQueued); ok {
			// ユーザに教えてあげる
			http.Redirect(w, req, "/?message=job_already_queued", http.StatusFound)
			return nil
		}

		return err
	}

	http.Redirect(w, req, "/", http.StatusFound)

	return nil
}

func updateTeamIPAddr(team *Team, ipAddr string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	res, err := tx.Exec("UPDATE teams SET ip_address = ? WHERE id = ?", ipAddr, team.ID)
	if err == nil {
		var nRows int64
		nRows, err = res.RowsAffected()
		if err == nil && nRows > 1 {
			err = fmt.Errorf("RowsAffected was %#v (> 1)", nRows)
		}
	}
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func remoteAddr(req *http.Request) string {
	if addr := req.Header.Get("X-Forwarded-For"); len(addr) != 0 {
		return addr
	}
	return req.RemoteAddr
}

// 新しいジョブを取り出す。ジョブが無い場合は 204 を返す
// クライアントは定期的(3秒おきくらい)にリクエストしてジョブを確認する
func ServeNewJob(w http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodPost {
		return errHTTP(http.StatusMethodNotAllowed)
	}
	benchNode := req.FormValue("bench_node")
	benchGroup := req.FormValue("bench_group")

	nodeAddr := remoteAddr(req)
	benchmarkNodesMtx.Lock()
	node, ok := benchmarkNodes[nodeAddr]
	benchmarkNodesMtx.Unlock()

	if !ok {
		node = BenchmarkNode{
			Name:   benchNode,
			Group:  benchGroup,
			IPAddr: nodeAddr,
		}
	}

	node.State = "Waiting"
	node.LastAccess = time.Now()

	defer func() {
		benchmarkNodesMtx.Lock()
		benchmarkNodes[nodeAddr] = node
		benchmarkNodesMtx.Unlock()
	}()

	j, err := dequeueJob(node)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return nil
	}
	if j == nil {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
	b, err := json.Marshal(j)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return nil
	}

	node.State = fmt.Sprintf("Attacking %s JobID=%d TeamID=%d", j.IPAddrs, j.ID, j.TeamID)

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
	return nil
}

func ServePostResult(w http.ResponseWriter, req *http.Request) error {
	if req.Method != http.MethodPost {
		http.Error(w, "Method Not Allowd", http.StatusMethodNotAllowed)
		return nil
	}

	aborted := req.URL.Query().Get("aborted") == "yes"
	jobID, err := strconv.Atoi(req.URL.Query().Get("jobid"))
	if err != nil {
		return err
	}

	nodeAddr := remoteAddr(req)
	benchmarkNodesMtx.Lock()
	node, ok := benchmarkNodes[nodeAddr]
	if ok {
		node.State = "DoneJob"
		if aborted {
			node.State = "Aborted"
		}
		node.LastAccess = time.Now()
		benchmarkNodes[nodeAddr] = node
	}
	benchmarkNodesMtx.Unlock()

	hasResult := false
	var res BenchResult
	var log string

	if req.Body == nil {
		http.Error(w, "Please send a request body", http.StatusBadRequest)
		return nil
	}

	if file, _, err := req.FormFile("result"); err == nil {
		defer file.Close()
		err := json.NewDecoder(file).Decode(&res)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return nil
		}
		hasResult = true
	}

	if file, _, err := req.FormFile("log"); err == nil {
		defer file.Close()
		b, err := ioutil.ReadAll(file)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return nil
		}
		log = string(b)
	}

	if hasResult {
		err := doneJob(&res, log)
		if err != nil {
			return err
		}
	} else {
		result := `{"reason":"Failed to decode result json"}`
		err := abortJob(jobID, result, log)
		if err != nil {
			return err
		}
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success":true}`)
	return nil
}
