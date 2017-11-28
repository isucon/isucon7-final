package main

import (
	"fmt"
	"math/rand"
	"sync"
)

var genDebugRoomName = false

var genRoom = struct {
	mtx  sync.Mutex
	cnt  int
	tags map[string]string
}{
	tags: map[string]string{},
}

func genRandomRoomName(tag string) string {
	genRoom.mtx.Lock()
	defer genRoom.mtx.Unlock()

	genRoom.cnt++

	if genDebugRoomName {
		name := fmt.Sprint("bench", genRoom.cnt)
		genRoom.tags[name] = tag
		return name
	} else {
		letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
		b := make([]rune, 20)
		for i := range b {
			b[i] = letterRunes[rand.Intn(len(letterRunes))]
		}
		name := "ROOM" + string(b)
		genRoom.tags[name] = tag
		return name
	}
}

func getRoomNameByTag(tag string) []string {
	genRoom.mtx.Lock()
	defer genRoom.mtx.Unlock()

	var res []string
	for room, rtag := range genRoom.tags {
		if tag == rtag {
			res = append(res, room)
		}
	}
	return res
}
