package main

import (
	"context"
	"fmt"
	"hash"
	"hash/fnv"
	"log"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"bench/counter"

	"github.com/gorilla/websocket"
)

var (
	ClientWriteTimeout = time.Second

	// GameResponse または GameStatus を受信する間隔
	ClientReadTimeout = time.Second

	// GameRequest 送信完了後 対応する GameResponse を受信するまでの時間
	ClientRequestTimeout = time.Second
)

// request and client id generator
var (
	requestCount int64
	clientCount  int64
)

func genRequestID() int64 {
	return atomic.AddInt64(&requestCount, 1)
}

func genClientID() int {
	return int(atomic.AddInt64(&clientCount, 1))
}

// error collection
var (
	clientLastError   atomic.Value
	clientFormatError atomic.Value
)

type clientError struct {
	t     time.Time
	err   error
	param interface{}
}

func (e *clientError) Error() string {
	return fmt.Sprintf("%v %v (%+v)", e.t, e.err, e.param)
}

func clearRecentClientError() {
	clientLastError.Store(&clientError{err: nil})
}

func getRecentClientError() *clientError {
	err := clientLastError.Load()
	if err != nil && err.(*clientError).err != nil {
		return err.(*clientError)
	}
	return nil
}

func getFormatError() error {
	err := clientFormatError.Load()
	if err != nil {
		return err.(error)
	}
	return nil
}

func onError(err error, param interface{}) error {
	if _, ok := err.(*clientError); ok {
		return err
	}

	cerr := &clientError{time.Now(), err, param}
	clientLastError.Store(cerr)
	return cerr
}

// websocket client wrapper
type client struct {
	id       int
	conn     *websocket.Conn
	roomName string
	wsAddr   string
	writeMtx sync.Mutex

	mtx      sync.Mutex
	status   *GameStatus
	updated  time.Time
	callback map[int]func(GameResponse)

	hasher    hash.Hash64
	closeOnce sync.Once
}

func (c *client) Start(ctx context.Context, room, wsAddr string) error {
	c.id = genClientID()
	c.roomName = room
	c.wsAddr = wsAddr
	c.callback = map[int]func(GameResponse){}
	c.hasher = fnv.New64a()
	c.closeOnce = sync.Once{}

	// TODO リクエストヘッダ, レスポンスは見なくても良いか?
	conn, _, err := websocket.DefaultDialer.Dial(wsAddr, nil)
	if err != nil {
		return err
	}
	counter.IncKey("client-open|" + room)
	c.conn = conn

	// 最初の一回のStatusを待つ
	c.conn.SetReadDeadline(time.Now().Add(ClientReadTimeout))
	_, err = c.read()
	if err != nil {
		c.Close()
		return onError(err, "read1")
	}

	c.mtx.Lock()
	hasStatus := c.status != nil
	c.mtx.Unlock()

	if !hasStatus {
		c.Close()
		return onError(fmt.Errorf("接続後最初 GameStatus 取得に失敗"), "")
	}

	go func() {
		defer c.Close()
		for {
			if ctx.Err() != nil {
				return
			}
			c.conn.SetReadDeadline(time.Now().Add(ClientReadTimeout))
			_, err := c.read()
			if err != nil {
				onError(err, "read2")
				return
			}
		}
	}()

	return nil
}

func (c *client) Close() error {
	c.closeOnce.Do(func() {
		counter.IncKey("client-close|" + c.roomName)
	})
	return c.conn.Close()
}

func (c *client) AddIsu(isu string, t int64) (GameRequest, GameResponse, error) {
	req := GameRequest{
		RequestID: int(genRequestID()),
		Action:    "addIsu",
		Time:      t,
		Isu:       isu,
	}
	res, err := c.doRequest(&req)
	return req, res, err
}

func (c *client) BuyItem(itemID, countBought int, t int64) (GameRequest, GameResponse, error) {
	req := GameRequest{
		RequestID:   int(genRequestID()),
		Action:      "buyItem",
		Time:        t,
		ItemID:      itemID,
		CountBought: countBought,
	}
	res, err := c.doRequest(&req)
	return req, res, err
}

func (c *client) doRequest(req *GameRequest) (GameResponse, error) {
	logOnRequest(c.roomName, &GameRequestLog{
		GameRequest: req,
		ClientID:    c.id,
		ClientTime:  time.Now(),
	})

	done := make(chan struct{})
	reqID := int(req.RequestID)
	res := GameResponse{}

	c.mtx.Lock()
	c.callback[reqID] = func(r GameResponse) {
		res = r
		close(done)
	}
	c.mtx.Unlock()

	defer func() {
		c.mtx.Lock()
		delete(c.callback, reqID)
		c.mtx.Unlock()
	}()

	c.writeMtx.Lock()
	c.conn.SetWriteDeadline(time.Now().Add(ClientWriteTimeout))
	err := c.conn.WriteJSON(req)
	c.writeMtx.Unlock()

	if err != nil {
		return GameResponse{}, onError(err, req)
	}

	select {
	case <-time.After(ClientRequestTimeout):
		c.Close()
		return GameResponse{}, onError(fmt.Errorf("request timeout"), req)
	case <-done:
		if res.IsSuccess {
			if req.Action == "addIsu" {
				counter.IncKey("client-addisu-ok|" + c.roomName)
				counter.IncKey("addisu-ok")
			} else if req.Action == "buyItem" {
				counter.IncKey("client-buyitem-ok|" + c.roomName)
				counter.IncKey("buyitem-ok")
			}
		} else {
			if req.Action == "addIsu" {
				counter.IncKey("client-addisu-ng|" + c.roomName)
			} else if req.Action == "buyItem" {
				counter.IncKey("client-buyitem-ng|" + c.roomName)
			}
		}
		return res, nil
	}
}

func (c *client) GetStatus() GameStatus {
	var s GameStatus

	c.mtx.Lock()
	if c.status != nil {
		s = *c.status
	}
	c.mtx.Unlock()
	return s
}

func (c *client) read() (interface{}, error) {
	_, p, err := c.conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	recvTime := time.Now()

	res, err := decodeReponseJson(p, c.hasher)
	if err != nil {
		clientFormatError.Store(fmt.Errorf("room %v にて Jsonデコードに失敗 %v", c.roomName, err))
		return nil, err
	}

	switch v := res.(type) {
	case *GameResponse:
		err = validateGameResponseFormat(v)
		if err != nil {
			clientFormatError.Store(fmt.Errorf("room %v にて %v", c.roomName, err))
			return nil, err
		}

		logOnResponse(c.roomName, &GameResponseLog{
			GameResponse: v,
			ClientID:     c.id,
			ClientTime:   recvTime,
		})

		c.mtx.Lock()
		f, ok := c.callback[v.RequestID]
		c.mtx.Unlock()
		if ok {
			f(*v)
		} else {
			log.Println("Unknown response", v)
		}
		return v, err
	case *GameStatus:
		err = validateGameStatusFormat(v)
		if err != nil {
			clientFormatError.Store(fmt.Errorf("room %v にて %v", c.roomName, err))
			return nil, err
		}

		logOnStatus(c.roomName, &GameStatusLog{
			GameStatus: v,
			ClientID:   c.id,
			ClientTime: recvTime,
		})

		c.mtx.Lock()
		c.status = v
		c.updated = recvTime
		c.mtx.Unlock()
		return v, err
	}

	return nil, fmt.Errorf("Jsonのデコードに失敗しました")
}

func (c *client) AfterDefault() int64 {
	return c.Now() + 800
}

func (c *client) After(dt int64) int64 {
	return c.Now() + dt
}

func (c *client) Now() int64 {
	var t int64
	var u time.Time

	c.mtx.Lock()
	if c.status != nil {
		t = c.status.Time
		u = c.updated
	}
	c.mtx.Unlock()

	if t == 0 {
		return 0
	}
	return t + time.Since(u).Nanoseconds()/1000000
}

func (c *client) WaitUntil(ctx context.Context, t int64) {
	ms := (t - c.Now())
	if ms <= 0 {
		return
	}
	c.Wait(ctx, ms)
}

func (c *client) Wait(ctx context.Context, ms int64) {
	d := time.Duration(ms) * time.Millisecond
	if time.Minute < d {
		log.Println("WARN client.Wait too large ms ", ms)
	}

	select {
	case <-time.After(d):
		return
	case <-ctx.Done():
		return
	}
}

// OnSale の Time を取得 無い場合は -1 を返却する
func (c *client) OnSaleTime(itemID int) int64 {
	for _, x := range c.GetStatus().OnSale {
		if x.ItemID == itemID {
			return x.Time
		}
	}
	return -1
}

func (c *client) CountBuilt(itemID int) int {
	for _, x := range c.GetStatus().Items {
		if x.ItemID == itemID {
			return x.CountBuilt
		}
	}
	return -1
}

func (c *client) CountBought(itemID int) int {
	for _, x := range c.GetStatus().Items {
		if x.ItemID == itemID {
			return x.CountBought
		}
	}
	return -1
}

func (c *client) GetPrice(m mItem) *big.Int {
	return m.GetPrice(c.CountBought(m.ItemID) + 1)
}
