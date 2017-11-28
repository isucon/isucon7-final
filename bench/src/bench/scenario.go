package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// load scenario

// 乱数個の椅子をaddIsuし続ける
func LoadAddIsu(ctx context.Context, room, wsAddr string) error {
	c := new(client)
	err := c.Start(ctx, room, wsAddr)
	if err != nil {
		return err
	}
	defer c.Close()

	for {
		s := genRandomNumberString(rand.Intn(50) + 1)
		_, _, err := c.AddIsu(s, c.AfterDefault())
		if err != nil {
			return err
		}

		// ユーザ数が増えることにメリットを与えるため, 1ユーザで頑張りすぎない
		time.Sleep(time.Millisecond * time.Duration(50+rand.Intn(10)))
	}
}

// 購入可能なアイテムIDの一番大きいアイテムを買う
func LoadBuyItem(ctx context.Context, room, wsAddr string) error {
	c := new(client)
	err := c.Start(ctx, room, wsAddr)
	if err != nil {
		return err
	}
	defer c.Close()

	for {
		for i := len(itemIDs) - 1; 0 <= i; i-- {
			mitem := mItems[itemIDs[i]]
			if c.OnSaleTime(mitem.ItemID) != 0 {
				continue
			}

			_, _, err := c.BuyItem(mitem.ItemID, c.CountBought(mitem.ItemID), c.AfterDefault())
			if err != nil {
				return err
			}
			break

			// 安いアイテムばかり高速に買ってしまわないように適度なsleep
			time.Sleep(time.Millisecond * 10)
		}
	}
}

func RSLoadIikanji(ctx context.Context, para chan struct{}) error {
	roomName := genRandomRoomName("load")
	wsAddr, err := resolveWsAddr(roomName)
	if err != nil {
		return err
	}

	addClient := int64(0)
	buyClient := int64(0)

	buyClientCh := make(chan struct{}, 1000)
	buyClientCh <- struct{}{}

	for {
		select {
		case <-para:
			go func() {
				err := LoadAddIsu(ctx, roomName, wsAddr)
				if err == nil {
					para <- struct{}{}
				}
				atomic.AddInt64(&addClient, -1)
			}()

			time.Sleep(10 * time.Millisecond)
		case <-buyClientCh:
			atomic.AddInt64(&buyClient, 1)
			go func() {
				LoadBuyItem(ctx, roomName, wsAddr)

				atomic.AddInt64(&buyClient, -1)
				select {
				case buyClientCh <- struct{}{}:
				default:
				}
			}()

			time.Sleep(10 * time.Millisecond)
		case <-ctx.Done():
			return nil
		}
	}
}

// PreTest

func PreTestIndexPage(ctx context.Context) error {
	url := "http://" + getRemoteAddr()
	log.Println("PreTestIndexPage", url)
	res, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("GETリクエストに失敗しました. %v", url)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return fmt.Errorf("期待していないステータスコード. %v", url)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return fmt.Errorf("ページのHTMLがパースできませんでした")
	}

	if !strings.Contains(doc.Find("head > title").Text(), "Chair Constructor Online") {
		return fmt.Errorf("トップページの title が Chair Constructor Online ではありません")
	}

	var (
		phinajs bool
		gamejs  bool
		guijs   bool
	)
	doc.Find("script").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		value := s.AttrOr("src", "")
		if !phinajs {
			phinajs = strings.Contains(value, "phina.js")
		}
		if !gamejs {
			gamejs = strings.Contains(value, "game.js")
		}
		if !guijs {
			guijs = strings.Contains(value, "gui.js")
		}
		return true
	})

	if !phinajs {
		return fmt.Errorf("トップページで phina.js が読み込まれていません。")
	}
	if !gamejs {
		return fmt.Errorf("トップページで game.js が読み込まれていません。")
	}
	if !guijs {
		return fmt.Errorf("トップページで gui.js が読み込まれていません。")
	}

	return nil
}

func PreTestStaticFile(ctx context.Context) error {
	for _, sf := range StaticFiles {
		url := "http://" + getRemoteAddr() + sf.Path
		log.Println("PreTestStaticFile", url)

		res, err := httpClient.Get(url)
		if err != nil {
			return fmt.Errorf("GETリクエストに失敗しました. %v", url)
		}
		defer res.Body.Close()

		if res.StatusCode != 200 {
			return fmt.Errorf("期待していないステータスコード. %v", url)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("レスポンスの取得に失敗 %v", url)
		}

		hash := md5.Sum(body)
		if hex.EncodeToString(hash[:]) != sf.Hash {
			return fmt.Errorf("静的ファイルの内容が一致していません %v", url)
		}
	}
	return nil
}

// wsのアドレスを取得するAPIの挙動を確認する.
func PreTestRoomAddr(ctx context.Context) error {
	for _, remote := range remoteAddrs {
		roomName := genRandomRoomName("preTest")
		// 適切な構造のJSONが返れば正しいものとする
		var x struct {
			Host string `json:"host"`
			Path string `json:"path"`
		}
		var url string

		// 空ではない部屋名
		url = fmt.Sprintf("http://%v/room/%v", remote, roomName)

		res1, err := httpClient.Get(url)
		if err != nil {
			return fmt.Errorf("GETリクエストに失敗しました. %v", url)
		}
		defer res1.Body.Close()

		bytes, err := ioutil.ReadAll(res1.Body)
		if err != nil {
			return fmt.Errorf("レスポンスの取得に失敗しました. %v", url)
		}
		err = json.Unmarshal(bytes, &x)
		if err != nil {
			return fmt.Errorf("JSONが正常に読み取れません. %v", url)
		}

		// 空の部屋名が許されるか
		url = fmt.Sprintf("http://%v/room/", remote)

		res2, err := httpClient.Get(url)
		if err != nil {
			return fmt.Errorf("GETリクエストに失敗しました. %v", url)
		}
		defer res2.Body.Close()

		bytes, err = ioutil.ReadAll(res2.Body)
		if err != nil {
			return fmt.Errorf("レスポンスの取得に失敗しました. %v", url)
		}
		err = json.Unmarshal(bytes, &x)
		if err != nil {
			return fmt.Errorf("JSONが正常に読み取れません. %v", url)
		}
	}
	return nil
}

func PreTestAddIsu(ctx context.Context) error {
	roomName := genRandomRoomName("preTest")
	wsAddr, err := resolveWsAddr(roomName)
	if err != nil {
		return err
	}

	c := new(client)
	err = c.Start(ctx, roomName, wsAddr)
	if err != nil {
		return fmt.Errorf("Room %v の接続に失敗しました. %v", roomName, err)
	}
	defer c.Close()

	// 近未来
	reqTime := c.After(999)
	_, res, err := c.AddIsu("11111111111111000000", reqTime)
	if err != nil {
		return fmt.Errorf("Room %v にて addIsu のリクエストに失敗しました. %v", roomName, err)
	}
	if !res.IsSuccess {
		return fmt.Errorf("Room %v にて addIsu が成功しませんでした. request_id = %v", roomName, res.RequestID)
	}

	// 過去
	reqTime = c.After(-1)
	_, res, err = c.AddIsu("11111111111111000000", reqTime)
	if err != nil {
		return fmt.Errorf("Room %v にて addIsu のリクエストに失敗しました. %v", roomName, err)
	}
	if res.IsSuccess {
		return fmt.Errorf("Room %v にて 過去に対する addIsu が成功しました. request_id = %v", roomName, res.RequestID)
	}

	return nil
}

func PreTestAddIsuMulti(ctx context.Context) error {
	roomName := genRandomRoomName("preTest")
	wsAddr, err := resolveWsAddr(roomName)
	if err != nil {
		return err
	}

	c1 := new(client)
	err = c1.Start(ctx, roomName, wsAddr)
	if err != nil {
		return fmt.Errorf("Room %v の接続に失敗しました. %v", roomName, err)
	}
	defer c1.Close()

	c2 := new(client)
	err = c2.Start(ctx, roomName, wsAddr)
	if err != nil {
		return fmt.Errorf("Room %v の接続に失敗しました. %v", roomName, err)
	}
	defer c2.Close()

	amount := "12345678901234500"
	addt := c1.After(999)

	var err1, err2 error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		_, res, err := c1.AddIsu(amount, addt)
		if err != nil {
			err1 = fmt.Errorf("Room %v にて addIsu のリクエストに失敗しました. %v", roomName, err)
			return
		}
		if !res.IsSuccess {
			err1 = fmt.Errorf("Room %v にて addIsu が成功しませんでした. request_id = %v", roomName, res.RequestID)
			return
		}
	}()

	go func() {
		defer wg.Done()

		_, res, err := c2.AddIsu(amount, addt)
		if err != nil {
			err2 = fmt.Errorf("Room %v にて addIsu のリクエストに失敗しました. %v", roomName, err)
			return
		}
		if !res.IsSuccess {
			err2 = fmt.Errorf("Room %v にて addIsu が成功しませんでした. request_id = %v", roomName, res.RequestID)
			return
		}
	}()

	wg.Wait()

	if err1 != nil {
		return err1
	}

	if err2 != nil {
		return err2
	}

	// addIsu が終わるまで待つ

	c1.WaitUntil(ctx, addt)
	c2.WaitUntil(ctx, addt)

	for i := 0; i < 30; i++ {
		st1 := c1.GetStatus()
		st2 := c2.GetStatus()

		log.Println("waiting addIsu", st1.Schedule[0].Time, st2.Schedule[0].Time)
		if addt <= st1.Schedule[0].Time && addt <= st2.Schedule[0].Time {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 検証はgamelogに任せる

	return nil
}

func PreTestBuyItem(ctx context.Context) error {
	check := func(mitem mItem) error {
		roomName := genRandomRoomName("preTest")
		wsAddr, err := resolveWsAddr(roomName)
		if err != nil {
			return err
		}

		c := new(client)
		err = c.Start(ctx, roomName, wsAddr)
		if err != nil {
			return fmt.Errorf("Room %v の接続に失敗しました. %v", roomName, err)
		}
		defer c.Close()

		// ちょうど必要なだけ作る
		addt := c.AfterDefault()
		_, res, err := c.AddIsu(fmt.Sprint(mitem.GetPrice(1)), addt)
		if err != nil {
			return fmt.Errorf("Room %v にて addIsu のリクエストに失敗しました. %v", roomName, err)
		}
		if !res.IsSuccess {
			return fmt.Errorf("Room %v にて addIsu が成功しませんでした. request_id = %v", roomName, res.RequestID)
		}

		c.WaitUntil(ctx, addt)
		for i := 0; i < 30; i++ {
			st := c.GetStatus()
			log.Println("waiting addIsu", st.Schedule[0].Time)
			if addt < st.Schedule[0].Time {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		// 建造
		buyt := c.AfterDefault()
		_, res, err = c.BuyItem(mitem.ItemID, 0, buyt)
		if err != nil {
			return fmt.Errorf("Room %v にて buyItem のリクエストに失敗しました. %v", roomName, err)
		}
		if !res.IsSuccess {
			return fmt.Errorf("Room %v にて buyItem が成功しませんでした. request_id = %v item_id = %v", roomName, res.RequestID, mitem.ItemID)
		}

		// buyItem が終わるまで待つ

		c.WaitUntil(ctx, buyt)
		for i := 0; i < 30; i++ {
			st := c.GetStatus()
			log.Println("waiting buyItem", st.Schedule[0].Time)
			if buyt < st.Schedule[0].Time {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		// 検証はgamelogに任せる

		return nil
	}

	// 全アイテムをチェックしていたが、Waitがあるため並列に投げないと時間かかりすぎる
	// 並列に投げると初期実装がタイムアウトでFailしてしまう可能性があるのでランダムに1個チェックするようにした
	// PreTestのランダム性排除するため ItemID = 8 のみチェックするようにした
	item8 := mItems[8]
	err := check(item8)
	if err != nil {
		return err
	}

	return nil
}

func PreTestBuyNotEnough(ctx context.Context) error {
	check := func(mitem mItem) error {
		roomName := genRandomRoomName("preTest")
		wsAddr, err := resolveWsAddr(roomName)
		if err != nil {
			return err
		}

		c := new(client)
		err = c.Start(ctx, roomName, wsAddr)
		if err != nil {
			return err
		}
		defer c.Close()

		_, res, err := c.BuyItem(mitem.ItemID, 1, c.AfterDefault())
		if err != nil {
			return fmt.Errorf("Room %v にて buyItem のリクエストに失敗しました. %v", roomName, err)
		}
		if res.IsSuccess {
			return fmt.Errorf("Room %v にて Isu が足りないにもかかわらず buyItem が成功しました. request_id = %v item_id = %v", roomName, res.RequestID, mitem.ItemID)
		}

		return nil
	}

	// 全アイテムをチェックしていたが、Waitがあるため並列に投げないと時間かかりすぎる
	// 並列に投げると初期実装がタイムアウトでFailしてしまう可能性があるのでランダムに1個チェックするようにした
	mitem := randomCheapItem()
	err := check(mitem)
	if err != nil {
		return err
	}

	return nil
}
