package client

import (
	"fmt"
	"golang.org/x/net/context"
	"os"
	"testing"
	"time"
)

var (
	c *Client
)

// go test example:
// `LAINLET_ADDR=192.168.77.21:9001 go test -v`
func init() {
	addr := os.Getenv("LAINLET_ADDR")
	c = New(addr)
}

func TestGet(t *testing.T) {
	if data, err := c.Get("/coreinfowatcher/?appname=console", time.Second*10); err != nil {
		t.Error(err)
	} else {
		t.Log(string(data))
	}
}

func TestWatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	resp, err := c.Watch("/vipconfwatcher/?heartbeat=3", ctx)
	if err != nil {
		t.Error(err)
	}
	for item := range resp {
		fmt.Println("Id: ", item.Id)
		fmt.Println("Event: ", item.Event)
		fmt.Println("Data: ", string(item.Data))
		fmt.Println("=========================")
	}
}
