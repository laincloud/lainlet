package etcd

import (
	"fmt"
	"golang.org/x/net/context"
	"github.com/laincloud/lainlet/store"
	"os"
	"strings"
	"testing"
	"time"
)

var (
	testStore *Etcd
)

func init() {
	var (
		err error
		tmp store.Store
	)
	addr := os.Getenv("LAIN_ETCD_ADDR")
	fmt.Println(addr)
	tmp, err = New(strings.Split(addr, ","))
	if err != nil {
		panic(err)
	}
	testStore = tmp.(*Etcd)
	testStore.Put("/test/dir/hello", []byte("world"))
	testStore.Put("/test/dir/world", []byte("hello"))
}

func TestGet(t *testing.T) {
	s, err := testStore.Get("/test/dir/hello")
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(s.Key, string(s.Value))
}

func TestList(t *testing.T) {
	s, err := testStore.List("/test")
	if err != nil {
		t.Error(err)
	}
	for _, item := range s {
		t.Log(item)
	}
}

func TestGetTree(t *testing.T) {
	s, err := testStore.GetTree("/test")
	if err != nil {
		t.Error(err)
	}
	for _, kv := range s {
		t.Log(kv.Key, string(kv.Value))
	}
}

func TestWatch(t *testing.T) {
	ch, err := testStore.Watch("/test", context.Background(), true)
	if err != nil {
		t.Error(err)
	}
	go func() {
		time.Sleep(time.Second)
		for i := 0; i < 10; i++ {
			testStore.Put("/test/dir/hello", []byte(fmt.Sprintf("test-%d", i)))
		}
		testStore.Delete("/test/dir/hello", false)
	}()
	for item := range ch {
		t.Log(item.Action.String(), item.Data[0].Key, string(item.Data[0].Value))
	}
}

func TestWatchTree(t *testing.T) {
	ch, err := testStore.WatchTree("/test", context.Background())
	if err != nil {
		t.Error(err)
	}
	go func() {
		time.Sleep(time.Second)
		testStore.Put("/test/dir/hello", []byte("test1"))
		testStore.Put("/test/dir/one", []byte("tet"))
		testStore.Put("/test/dir/two", []byte("teasdfat"))
		testStore.Put("/test/dir/world", []byte("test1"))
		testStore.Delete("/test/world", false)
		testStore.Delete("/test/two", false)
		testStore.Delete("/test", true)
	}()
	for item := range ch {
		t.Log(item.Action.String(), item.Key)
		for _, kv := range item.Data {
			t.Log("    ", kv.Key, string(kv.Value))
		}
		t.Log("====================================")
	}
}
