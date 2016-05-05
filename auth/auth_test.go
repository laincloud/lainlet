package auth

import (
	"fmt"
	"github.com/mijia/sweb/log"
	"golang.org/x/net/context"
	"github.com/laincloud/lainlet/store"
	_ "github.com/laincloud/lainlet/store/etcd"
	"github.com/laincloud/lainlet/watcher"
	"os"
	"strings"
	"testing"
	"time"
)

var (
	s store.Store
	w *watcher.Watcher
)

func init() {
	log.EnableDebug()
	var err error
	addr := os.Getenv("LAIN_ETCD_ADDR")
	fmt.Println(addr)
	s, err = store.New("etcd", strings.Split(addr, ","))
	if err != nil {
		panic(err)
	}
	if err = Init(s, context.Background(), "127.0.0.1"); err != nil {
		panic(err)
	}
}

func TestPass(t *testing.T) {
	time.Sleep(time.Second)
	testList := [][2]string{
		{"127.0.0.1", "registry"},
		{"172.20.0.9", "registry"},
	}
	for _, item := range testList {
		if !Pass(item[0], item[1]) {
			t.Error(fmt.Errorf("%s can not pass %s", item[0], item[1]))
		}
	}
}
