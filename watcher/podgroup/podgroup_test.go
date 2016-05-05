package podgroup

import (
	"encoding/json"
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
	w, err = New(s, context.Background())
	if err != nil {
		panic(err)
	}
}

func TestGetAll(t *testing.T) {
	data, err := w.Get("*")
	if err != nil {
		t.Error(err)
	}
	content, err := json.Marshal(data)
	if err != nil {
		t.Error(err)
	}
	t.Log(string(content))
}

func TestWatch(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
	ch, err := w.Watch("backupctl", ctx)
	if err != nil {
		t.Error(err)
	}
	log.Infof("Not you can set the etcd by hands, check if there is anything output here!")
	fmt.Println("")
	for item := range ch {
		content, err := json.Marshal(item)
		if err != nil {
			t.Error(err)
		}
		log.Debugf(string(content))
	}
	time.Sleep(time.Second)
}
