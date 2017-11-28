package portal

import (
	"log"
	"net/http"
	"sort"
	"strings"
)

func ServeAdminPage(w http.ResponseWriter, r *http.Request) error {
	team, err := loadTeamFromSession(r)
	if err != nil {
		return err
	}
	if !team.IsAdmin() {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return nil
	}

	if r.Method == "POST" {
		log.Println(r.RequestURI, remoteAddr(r))
		err := r.ParseForm()
		if err != nil {
			return err
		}

		confirmed := r.FormValue("confirm") == "yes"

		for k, v := range r.Form {
			log.Println("key:", k)
			log.Println("val:", strings.Join(v, ""))

			switch k {
			case "info":
				infoText = strings.Join(v, "")
			case "status":
				if !confirmed {
					continue
				}
				for _, s := range v {
					switch s {
					case "notstarted":
						contestStatus = ContestStatusNotStarted
					case "started":
						contestStatus = ContestStatusStarted
					case "ended":
						contestStatus = ContestStatusEnded
					}
				}
			default:
			}
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
		return nil
	}

	type keyValue struct {
		Key   string
		Value string
	}

	keyValues := []keyValue{}

	state := ""
	switch GetContestStatus() {
	case ContestStatusNotStarted:
		state = "NotStarted"
	case ContestStatusStarted:
		state = "Started"
	case ContestStatusEnded:
		state = "Ended"
	}
	keyValues = append(keyValues, keyValue{"ContestState", state})

	nodes := []BenchmarkNode{}
	benchmarkNodesMtx.Lock()
	for _, node := range benchmarkNodes {
		nodes = append(nodes, node)
	}
	benchmarkNodesMtx.Unlock()

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].LastAccess.After(nodes[j].LastAccess)
	})

	return templates["admin.tmpl"].Execute(w,
		struct {
			viewParamsLayout
			Info             string
			ContestDayNumber int
			KeyValues        []keyValue
			Nodes            []BenchmarkNode
		}{
			viewParamsLayout{nil, contestDayNumber},
			infoText,
			contestDayNumber,
			keyValues,
			nodes,
		})
}

func ServeAdminServer(w http.ResponseWriter, r *http.Request) error {
	if t, err := loadTeamFromSession(r); err != nil {
		return err
	} else if t == nil || !t.IsAdmin() {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return nil
	}

	type server struct {
		ID       int
		Name     string
		LocalIP  string
		GlobalIP string
		Group    string
	}

	type team struct {
		ID       int
		Name     string
		Password string
		Category string
		Servers  []*server
		Group    string
	}

	teams := []*team{}
	teamByID := map[int]*team{}
	servers := []*server{}

	rows, err := db.Query("SELECT id, name, password, category, `group` FROM teams ORDER BY id ASC")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		t := team{}
		err := rows.Scan(&t.ID, &t.Name, &t.Password, &t.Category, &t.Group)
		if err != nil {
			return err
		}
		teams = append(teams, &t)
		teamByID[t.ID] = &t
	}
	if err := rows.Err(); err != nil {
		return err
	}

	cntTeamServer := 0
	rows, err = db.Query("SELECT id, name, team_id, local_ip, global_ip, `group` FROM servers ORDER BY id ASC")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		sv := server{}
		teamID := 0
		err := rows.Scan(&sv.ID, &sv.Name, &teamID, &sv.LocalIP, &sv.GlobalIP, &sv.Group)
		if err != nil {
			return nil
		}

		t, ok := teamByID[teamID]
		if ok {
			cntTeamServer++
			t.Servers = append(t.Servers, &sv)
		} else {
			servers = append(servers, &sv)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	return templates["admin-server.tmpl"].Execute(w,
		struct {
			viewParamsLayout
			CntTeamServer int
			Teams         []*team
			Servers       []*server
		}{
			viewParamsLayout{nil, contestDayNumber},
			cntTeamServer,
			teams,
			servers,
		})
}
