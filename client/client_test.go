package client

import (
	"fmt"
	"github.com/stretchr/testify/assert"
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

func TestAppName(t *testing.T) {
	data, _ := c.Get("/appname?ip=172.20.0.168", time.Second*10)
	assert.Equal(t, data, []byte("{\"appname\":\"ipaddr-client\"}"))
}

func TestAppWatcher(t *testing.T) {
	ctx,cancel := context.WithCancel(context.Background())
	resp, _ := c.Watch("/v2/appswatcher", ctx)
	go func() {
		time.Sleep(time.Second*5)
		cancel()
	}()
	for item := range resp {
		fmt.Println("Id: ", item.Id)
		fmt.Println("Event: ", item.Event)
		fmt.Println("Data: ", string(item.Data))
		fmt.Println("=========================")
	}

}

func TestBackupSpec(t *testing.T) {
	ctx,cancel := context.WithCancel(context.Background())
	resp, _ := c.Watch("/v2/backupspec", ctx)
	go func() {
		time.Sleep(time.Second*5)
		cancel()
	}()
	for item := range resp {
		fmt.Println("Id: ", item.Id)
		fmt.Println("Event: ", item.Event)
		fmt.Println("Data: ", string(item.Data))
		fmt.Println("=========================")
	}
}

func TestConfigWatcher(t *testing.T)  {
	ctx,cancel := context.WithCancel(context.Background())
	resp, _ := c.Watch("/v2/configwatcher", ctx)
	go func() {
		time.Sleep(time.Second*5)
		cancel()
	}()
	for item := range resp {
		fmt.Println("Id: ", item.Id)
		fmt.Println("Event: ", item.Event)
		fmt.Println("Data: ", string(item.Data))
		fmt.Println("=========================")
	}
}

func TestContainers(t *testing.T) {
	ctx,cancel := context.WithCancel(context.Background())
	resp, _ := c.Watch("/v2/containers", ctx)
	go func() {
		time.Sleep(time.Second*5)
		cancel()
	}()
	for item := range resp {
		fmt.Println("Id: ", item.Id)
		fmt.Println("Event: ", item.Event)
		fmt.Println("Data: ", string(item.Data))
		fmt.Println("=========================")
	}
}

func TestCoreInfoWatcher(t *testing.T) {
	ctx,cancel := context.WithCancel(context.Background())
	resp, _ := c.Watch("/v2/coreinfowatcher", ctx)
	go func() {
		time.Sleep(time.Second*5)
		cancel()
	}()
	for item := range resp {
		fmt.Println("Id: ", item.Id)
		fmt.Println("Event: ", item.Event)
		fmt.Println("Data: ", string(item.Data))
		fmt.Println("=========================")
	}
}

func TestDepends(t *testing.T){
	ctx,cancel := context.WithCancel(context.Background())
	resp, _ := c.Watch("/v2/depends", ctx)
	go func() {
		time.Sleep(time.Second*5)
		cancel()
	}()
	for item := range resp {
		fmt.Println("Id: ", item.Id)
		fmt.Println("Event: ", item.Event)
		fmt.Println("Data: ", string(item.Data))
		fmt.Println("=========================")
	}
}

func TestLocalSpecQuery(t *testing.T){
	resp, _ := c.Get("/v2/localspecquery?nodeip=172.20.0.168",time.Second*5)
	t.Log(string(resp))
}

func TestNodes(t *testing.T){
	ctx,cancel := context.WithCancel(context.Background())
	resp, _ := c.Watch("/v2/nodes", ctx)
	go func() {
		time.Sleep(time.Second*5)
		cancel()
	}()
	for item := range resp {
		fmt.Println("Id: ", item.Id)
		fmt.Println("Event: ", item.Event)
		fmt.Println("Data: ", string(item.Data))
		fmt.Println("=========================")
	}
}

func TestProcWatcher(t *testing.T){
	ctx,cancel := context.WithCancel(context.Background())
	resp, _ := c.Watch("/v2/procwatcher?appname=ipaddr-client", ctx)
	go func() {
		time.Sleep(time.Second*5)
		cancel()
	}()
	for item := range resp {
		fmt.Println("Id: ", item.Id)
		fmt.Println("Event: ", item.Event)
		fmt.Println("Data: ", string(item.Data))
		fmt.Println("=========================")
	}
}

func TestProxyWatcher(t *testing.T) {
	ctx,cancel := context.WithCancel(context.Background())
	resp, _ := c.Watch("/v2/proxywatcher?appname=ipaddr-client", ctx)
	go func() {
		time.Sleep(time.Second*5)
		cancel()
	}()
	for item := range resp {
		fmt.Println("Id: ", item.Id)
		fmt.Println("Event: ", item.Event)
		fmt.Println("Data: ", string(item.Data))
		fmt.Println("=========================")
	}
}

func TestRebellionLocalProcs(t *testing.T){
	ctx,cancel := context.WithCancel(context.Background())
	resp, _ := c.Watch("/v2/rebellion/localprocs?appname=ipaddr-client", ctx)
	go func() {
		time.Sleep(time.Second*5)
		cancel()
	}()
	for item := range resp {
		fmt.Println("Id: ", item.Id)
		fmt.Println("Event: ", item.Event)
		fmt.Println("Data: ", string(item.Data))
		fmt.Println("=========================")
	}
}

func TestWebRouterWebProcs(t *testing.T){
	ctx,cancel := context.WithCancel(context.Background())
	resp, _ := c.Watch("/v2/webrouter/webprocs?appname=ipaddr-client", ctx)
	go func() {
		time.Sleep(time.Second*5)
		cancel()
	}()
	for item := range resp {
		fmt.Println("Id: ", item.Id)
		fmt.Println("Event: ", item.Event)
		fmt.Println("Data: ", string(item.Data))
		fmt.Println("=========================")
	}
}
