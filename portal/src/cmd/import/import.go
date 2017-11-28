package main

// ./bin/import -target servers -dsn-base 'root:@(localhost)' < data/dummy-servers.tsv
// ./bin/import -target teams -dsn-base 'root:@(localhost)' < data/dummy-teams.tsv

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

var (
	target     = flag.String("target", "", "import target (teams, servers)")
	dsnBase    = flag.String("dsn-base", "root:@(localhost)", "`dsn` base address (w/o database name) for isu7fportal")
	dbNameDay0 = flag.String("db-day0", "isu7fportal_day0", "`database` name for day 0")
	dbNameDay1 = flag.String("db-day1", "isu7fportal_day1", "`database` name for day 1")
)

var (
	db0, db1 *sql.DB
)

const (
	operatorTeamID   = 9999
	operatorPassword = "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	flag.Parse()

	options := "?charset=utf8mb4&parseTime=true&loc=Asia%2FTokyo&time_zone='Asia%2FTokyo'"

	var err error
	db0, err = sql.Open("mysql", *dsnBase+"/"+*dbNameDay0+options)
	if err != nil {
		log.Fatal(err)
	}
	db1, err = sql.Open("mysql", *dsnBase+"/"+*dbNameDay1+options)
	if err != nil {
		log.Fatal(err)
	}

	for _, db := range []*sql.DB{db1} {
		_, err = db.Exec("SET SESSION sql_mode='TRADITIONAL,NO_AUTO_VALUE_ON_ZERO,ONLY_FULL_GROUP_BY'")
		if err != nil {
			log.Fatal(err)
		}
	}

	switch *target {
	case "teams":
		importTeams()
	case "servers":
		importServers()
	default:
		log.Fatal("invalid target")
	}
}

func importTeams() {
	s := bufio.NewScanner(os.Stdin)
	s.Scan() // drop first line
	for s.Scan() {
		parts := strings.Split(s.Text(), "\t")
		// (0)カテゴリ (1)暫定No (2)チーム名 (3)人数 (4)代表者 (5)パスワード (6)グループ
		var (
			category  string
			teamID    int64
			name      string = parts[2]
			password  string = parts[5]
			groupName string = parts[6]
			err       error
		)
		// 人数と代表者は無視

		switch parts[0] {
		case "一般":
			category = "general"
		case "学生":
			category = "students"
		default:
			log.Fatalf("unknown category: %q", parts[2])
		}

		teamID, err = strconv.ParseInt(parts[1], 10, 0)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Inserting into db1 id=%#v group=%#v name=%#v password=%#v category=%#v", teamID, groupName, name, password, category)
		_, err = db1.Exec("REPLACE INTO teams (id, `group`, name, password, category) VALUES (?, ?, ?, ?, ?)", teamID, groupName, name, password, category)
		if err != nil {
			log.Fatal(err)
		}
	}

	// day0 はダミーデータで埋める
	for n := 1; n <= 20; n++ {
		var category string
		if n%2 == 0 {
			category = "general"
		} else {
			category = "students"
		}
		teamID := 1000 + n
		groupName := fmt.Sprintf("GROUP%d", (teamID+1)/2)
		_, err := db0.Exec("REPLACE INTO teams (id, `group`, name, password, category) VALUES (?, ?, ?, ?, ?)", teamID, groupName, fmt.Sprintf("ダミーチーム%d", n), fmt.Sprintf("dummy-pass-%d", n), category)
		if err != nil {
			log.Fatal(err)
		}
	}

	// 運営アカウントいれる
	for _, db := range []*sql.DB{db0, db1} {
		_, err := db.Exec("REPLACE INTO teams (id, `group`, name, password, category) VALUES (?, ?, ?, ?, ?)", operatorTeamID, "STAFF", "運営", operatorPassword, "official")
		if err != nil {
			log.Fatal(err)
		}
	}

	/* FIXME : 本選用にバリデーション
	for _, p := range []struct {
		day   int
		db    *sql.DB
		count int
	}{{1, db1, count1}} {
		var c int
		err := p.db.QueryRow("SELECT COUNT(*) FROM teams").Scan(&c)
		if err != nil {
			log.Fatal(err)
		}

		c-- // 運営アカウントの分

		if c != p.count {
			log.Fatalf("team count for day %d is incorrect!! expected: %d actual: %d", p.day, p.count, c)
		} else {
			log.Printf("#teams for day %d: %d", p.day, p.count)
		}
	}
	*/
}

func importServers() {
	s := bufio.NewScanner(os.Stdin)
	s.Scan() // drop first line
	for s.Scan() {
		parts := strings.Split(s.Text(), "\t")

		serverID, err := strconv.ParseInt(parts[0], 10, 0)
		if err != nil {
			log.Fatal(err)
		}

		teamID, err := strconv.ParseInt(parts[1], 10, 0)
		if err != nil {
			log.Fatal(err)
		}

		serverName := strings.TrimSpace(parts[2])
		globalIP := strings.TrimSpace(parts[3])
		localIP := strings.TrimSpace(parts[4])
		group := strings.TrimSpace(parts[5])

		_, err = db1.Exec("REPLACE INTO servers (id, name, team_id, local_ip, global_ip, `group`) VALUES (?, ?, ?, ?, ?, ?)",
			serverID, serverName, teamID, localIP, globalIP, group)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("inserted id=%#v name=%#v team_id=%#v local_ip=%#v global_ip=%#v group=%#v", serverID, serverName, teamID, localIP, globalIP, group)
	}

	// day0 はダミーデータで埋める
	serverID := 1000
	for n := 1; n <= 20; n++ {
		for i := 0; i < 4; i++ {
			teamID := 1000 + n
			serverName := fmt.Sprintf("dummy-server-%d", serverID)
			groupName := fmt.Sprintf("GROUP%d", (teamID+1)/2)
			_, err := db0.Exec("REPLACE INTO servers (id, name, team_id, local_ip, global_ip, `group`) VALUES (?, ?, ?, ?, ?, ?)",
				serverID, serverName, teamID, fmt.Sprint("127.0.0.1"), fmt.Sprint("127.0.0.1"), groupName)
			if err != nil {
				log.Fatal(err)
			}
			serverID++
		}
	}
}
