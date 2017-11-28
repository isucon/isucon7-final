package main

import (
	"bench/counter"
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/profile"
)

var (
	benchDuration     = 1 * time.Minute
	preTestTimeout    = 15 * time.Second
	postTestTimeout   = 10 * time.Second
	loadLogs          []string
	loadLevel         int
	noLevelup         bool
	noCheckStaticFile bool
	preTestOnly       bool
	mItems            = map[int]mItem{}
	itemIDs           []int
	remoteAddrs       []string

	pprofPort  = 16060
	httpClient = http.Client{
		Timeout: 10 * time.Second,
	}
)

func getRemoteAddr() string {
	return remoteAddrs[rand.Intn(len(remoteAddrs))]
}

func randomItem() mItem {
	return mItems[itemIDs[rand.Intn(len(itemIDs))]]
}

func randomCheapItem() mItem {
	return mItems[itemIDs[rand.Intn(len(itemIDs)/2)]]
}

func printMetrics() {
	// room metrics
	log.Println("- Metrics -")
	m := counter.GetMap()
	rooms := getRoomNameByTag("load")

	type record struct {
		key int64
		msg string
	}

	msgs := []record{}
	for _, r := range rooms {
		open := m["client-open|"+r]
		closed := m["client-close|"+r]
		active := open - closed
		addok := m["client-addisu-ok|"+r]
		addng := m["client-addisu-ng|"+r]
		buyok := m["client-buyitem-ok|"+r]
		buyng := m["client-buyitem-ng|"+r]

		msg := fmt.Sprintf("%v Open:%v Close:%v Active:%v AddOK-NG:%v-%v BuyOK-NG:%v-%v",
			r, open, closed, active, addok, addng, buyok, buyng)
		msgs = append(msgs, record{key: open, msg: msg})
	}

	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].key > msgs[j].key
	})

	for _, msg := range msgs {
		log.Print(msg.msg)
	}

	for _, key := range []string{
		"hash-bin-hit",
		"hash-adding-hit",
		"hash-schedule-hit",
		"hash-items-hit",
		"hash-onsale-hit",
	} {
		log.Println(key, counter.GetKey(key))
	}

	if StrictCheckCacheConflict {
		for _, key := range []string{
			"hash-bin-conflict",
			"hash-adding-conflict",
			"hash-schedule-conflict",
			"hash-items-conflict",
			"hash-onsale-conflict",
		} {
			log.Println(key, counter.GetKey(key))
		}
	}
}

func resolveWsAddr(roomName string) (string, error) {
	url := fmt.Sprintf("http://%v/room/%v", getRemoteAddr(), roomName)
	log.Println(url)
	res, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var x struct {
		Host string `json:"host"`
		Path string `json:"path"`
	}
	err = json.Unmarshal(bytes, &x)
	if err != nil {
		return "", err
	}

	if x.Host == "" {
		x.Host = res.Request.Host
	}

	return fmt.Sprintf("ws://%v%v", x.Host, x.Path), nil
}

func requestInitialize(host string) error {
	url := fmt.Sprintf("http://%v/initialize", host)
	res, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 && res.StatusCode != 204 {
		return fmt.Errorf("期待していないステータスコード %v", res.StatusCode)
	}

	return nil
}

func benchmarkMain(ctx context.Context) {
	var chans []chan struct{}

	// ルームのユーザ数を増やす
	// 各ルーム5人ずつ入れるが max人到達時点でやめる
	addRoomUser := func(max int) {
		cnt := 0
		for i := 0; i < 5; i++ {
			for _, i := range rand.Perm(len(chans)) {
				c := chans[i]
				if 0 < max && max <= cnt {
					break
				}
				select {
				case c <- struct{}{}:
					cnt++
				default:
				}
			}
		}
	}

	// ルームを新しく作る
	addRoom := func() {
		c1 := make(chan struct{}, 1000)
		go RSLoadIikanji(ctx, c1)
		chans = append(chans, c1)
	}

	addRoom()
	addRoomUser(-1)

	beat := time.NewTicker(time.Second)
	defer beat.Stop()

	maxActive := int64(0)
	prevClose := int64(0)
	prevActive := int64(0)
	activeLimit := int64(-1)

	for {
		select {
		case <-beat.C:
			if getFormatError() != nil {
				return
			}

			printMetrics()

			if noLevelup {
				continue
			}

			err := getRecentClientError()
			hasRecentErr := err != nil && time.Since(err.t) < 5*time.Second

			open := counter.SumPrefix("client-open|")
			close := counter.SumPrefix("client-close|")
			active := open - close
			if maxActive < active {
				maxActive = active
			}

			if 5 < prevActive && float64(prevActive)*0.2 < float64(close-prevClose) {
				activeLimit = int64(float64(maxActive) * 0.6)
			}

			now := time.Now().Format("01/02 15:04:05")
			log.Println("Active:", active, "Limit:", activeLimit)

			if hasRecentErr {
				log.Println("RecentError", err)
			}
			if !hasRecentErr && activeLimit == -1 {
				msg := fmt.Sprintf("%v 接続数が増加します", now)
				loadLogs = append(loadLogs, msg)

				loadLevel++
				addRoom()
				addRoomUser(-1)
				log.Println(msg)
			} else if 0 < activeLimit && active < activeLimit {
				msg := fmt.Sprintf("%v 接続数が増加します (上限:%v)", now, activeLimit)
				loadLogs = append(loadLogs, msg)

				addRoomUser(int(activeLimit - active))
				log.Println(msg)
			}

			prevClose = close
			prevActive = active
		case <-ctx.Done():
			return
		}
	}
}

func preTest() error {
	ctx, cancel := context.WithTimeout(context.Background(), preTestTimeout)
	defer cancel()

	// preTest中だけタイムアウトを伸ばす
	var b1, b2, b3 = ClientRequestTimeout, ClientReadTimeout, ClientWriteTimeout
	ClientRequestTimeout, ClientReadTimeout, ClientWriteTimeout = 5*time.Second, 5*time.Second, 5*time.Second
	defer func() {
		ClientRequestTimeout, ClientReadTimeout, ClientWriteTimeout = b1, b2, b3
	}()

	if !noCheckStaticFile {
		err := PreTestIndexPage(ctx)
		if err != nil {
			return err
		}

		err = PreTestStaticFile(ctx)
		if err != nil {
			return err
		}
	}

	err := PreTestRoomAddr(ctx)
	if err != nil {
		return err
	}

	err = PreTestAddIsu(ctx)
	if err != nil {
		return err
	}

	err = PreTestAddIsuMulti(ctx)
	if err != nil {
		return err
	}

	err = PreTestBuyItem(ctx)
	if err != nil {
		return err
	}

	err = PreTestBuyNotEnough(ctx)
	if err != nil {
		return err
	}

	for _, room := range getRoomNameByTag("preTest") {
		err := ValidateGameLog(ctx, room, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func postTest() error {
	ctx, cancel := context.WithTimeout(context.Background(), postTestTimeout)
	defer cancel()

	var ret atomic.Value
	var wg sync.WaitGroup
	cpus := runtime.NumCPU()
	sem := make(chan struct{}, cpus)
	for i := 0; i < cpus; i++ {
		sem <- struct{}{}
	}

	for _, r := range getRoomNameByTag("load") {
		room := r
		select {
		case <-sem:
			wg.Add(1)
			go func() {
				defer func() {
					wg.Done()
					sem <- struct{}{}
				}()

				err := ValidateGameLog(ctx, room, false)
				if err != nil {
					ret.Store(fmt.Errorf("Room %v にて %v", room, err))
					cancel()
					return
				}
			}()
		case <-ctx.Done():
		}
	}

	wg.Wait()

	if ctx.Err() == context.DeadlineExceeded {
		log.Println("Validation Timeout")
	}

	err := ret.Load()
	if err != nil {
		return err.(error)
	}
	return nil
}

func startBenchmark() *BenchResult {
	result := new(BenchResult)
	result.StartTime = time.Now()
	defer func() {
		result.EndTime = time.Now()
	}()

	log.Println("requestInitialize()")
	err := requestInitialize(getRemoteAddr())
	if err != nil {
		result.Score = 0
		result.Message = fmt.Sprint("/initialize へのリクエストに失敗しました。", err)
		return result
	}
	log.Println("requestInitialize() Done")

	log.Println("preTest()")
	err = preTest()
	if getFormatError() != nil {
		err = getFormatError()
	}
	if err != nil {
		result.Score = 0
		result.Message = fmt.Sprint("負荷走行前のバリデーションに失敗しました。", err)
		return result
	}
	log.Println("preTest() Done")

	if preTestOnly {
		result.Message = fmt.Sprint("preTest passed.")
		return result
	}

	ctx, cancel := context.WithTimeout(context.Background(), benchDuration)
	defer cancel()

	// 負荷レベルを上げる条件に関わる為 PreTestのエラーを無視する
	clearRecentClientError()
	log.Println("benchmarkMain()")
	benchmarkMain(ctx)
	log.Println("benchmarkMain() Done")

	// ベンチ終わった瞬間の値を取っておく
	a := counter.GetKey("addisu-ok")
	b := counter.GetKey("buyitem-ok")
	log.Println(a, b)
	result.Logs = loadLogs

	err = getFormatError()
	if err != nil {
		result.Score = 0
		result.Message = fmt.Sprint("負荷走行中のバリデーションに失敗しました。", err)
		return result
	}

	err = postTest()
	if err != nil {
		result.Score = 0
		result.Message = fmt.Sprint("負荷走行後のバリデーションに失敗しました。", err)
		return result
	}

	result.Score = a + 10*b
	result.Pass = true
	result.LoadLevel = loadLevel
	result.Message = "ok"
	return result
}

func loadMasterData(dataPath string) {
	fp, err := os.Open(filepath.Join(dataPath, "m_item.tsv"))
	must(err)

	reader := csv.NewReader(fp)
	reader.Comma = '\t'

	mustInt := func(s string) int {
		v, err := strconv.Atoi(s)
		must(err)
		return v
	}

	mustInt64 := func(s string) int64 {
		v, err := strconv.ParseInt(s, 10, 64)
		must(err)
		return v
	}

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		must(err)
		m := mItem{
			ItemID: mustInt(row[0]),
			Power1: mustInt64(row[1]),
			Power2: mustInt64(row[2]),
			Power3: mustInt64(row[3]),
			Power4: mustInt64(row[4]),
			Price1: mustInt64(row[5]),
			Price2: mustInt64(row[6]),
			Price3: mustInt64(row[7]),
			Price4: mustInt64(row[8]),
		}
		mItems[m.ItemID] = m
		itemIDs = append(itemIDs, m.ItemID)
	}

	sort.Slice(itemIDs, func(i, j int) bool {
		return itemIDs[i] < itemIDs[j]
	})
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	log.SetPrefix("[isu7f-bench] ")

	var (
		workermode   bool
		portalUrl    string
		tempdir      string
		jobid        string
		dataPath     string
		remotes      string
		nolevelup    bool
		nostaticfile bool
		test         bool
		output       string
		debugname    bool
		saveprofile  bool
		memprofile   bool
		dumpgamelog  int
		strictcache  bool
		validatelog  string
	)

	flag.BoolVar(&workermode, "workermode", false, "workermode")
	flag.StringVar(&portalUrl, "portal", "http://localhost:8888", "portal site url (only used on workermode)")
	flag.StringVar(&tempdir, "tempdir", "", "path to temp dir")
	flag.StringVar(&jobid, "jobid", "", "job id")
	flag.StringVar(&dataPath, "data", "./data", "path to data directory")
	flag.StringVar(&remotes, "remotes", "localhost:5000", "remote addrs to benchmark")
	flag.BoolVar(&test, "test", false, "run pretest only")
	flag.BoolVar(&nolevelup, "nolevelup", false, "dont increase load level")
	flag.BoolVar(&nostaticfile, "nostaticfile", false, "dont check static file")
	flag.StringVar(&output, "output", "", "path to write result json")
	flag.BoolVar(&debugname, "debugname", false, "use benchN as room name")
	flag.BoolVar(&saveprofile, "profile", false, "save cpu profile into tmp direcotry")
	flag.BoolVar(&memprofile, "memprof", false, "save mem profile into tmp direcotry")
	flag.IntVar(&dumpgamelog, "dumpgamelog", 0, "save gamelog into tmp direcotry (1:if postTest failed 2:always)")
	flag.BoolVar(&strictcache, "strictcache", false, "compare cached json strictly")
	flag.StringVar(&validatelog, "validatelog", "", "path to gzipped gamelog to debug validation")
	flag.Parse()

	loadMasterData(dataPath)

	if workermode {
		runWorkerMode(tempdir, portalUrl)
		return
	}

	go func() {
		log.Println(http.ListenAndServe(fmt.Sprintf(":%d", pprofPort), nil))
	}()

	if saveprofile {
		defer profile.Start().Stop()
	}
	if memprofile {
		defer profile.Start(profile.MemProfile).Stop()
	}
	noLevelup = nolevelup
	noCheckStaticFile = nostaticfile
	preTestOnly = test
	genDebugRoomName = debugname
	saveGameLogDump = dumpgamelog
	StrictCheckCacheConflict = strictcache
	remoteAddrs = strings.Split(remotes, ",")

	if validatelog != "" {
		err := ValidateGameLogDump(validatelog)
		if err != nil {
			log.Println(err)
		}
		return
	}

	rand.Seed(time.Now().UnixNano())
	benchResult := startBenchmark()
	benchResult.IPAddrs = remotes
	benchResult.JobID = jobid

	benchResultJSON, err := json.Marshal(benchResult)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println(string(benchResultJSON))
	if output != "" {
		err := ioutil.WriteFile(output, benchResultJSON, 0644)
		if err != nil {
			log.Fatalln(err)
		}
		log.Println("result json saved to ", output)
	}
}
