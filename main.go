package main

import (
	"flag"
	"os"
	"os/signal"
	"strings"

	"github.com/laincloud/lainlet/api"
	"github.com/laincloud/lainlet/api/v2"
	"github.com/laincloud/lainlet/auth"
	"github.com/laincloud/lainlet/store"
	_ "github.com/laincloud/lainlet/store/etcd"
	_ "github.com/laincloud/lainlet/store/mocks"
	"github.com/mijia/sweb/log"
	"golang.org/x/net/context"
)

const (
	// the version value
	version = "2.0.4"
)

var (
	webAddr, storeType, storeAddr, ip string
	debug, v, noAuth                  bool
)

func init() {
	flag.StringVar(&webAddr, "web", "localhost:9000", "The address lainlet listen")
	flag.StringVar(&storeType, "type", "etcd", "The store type, such as etcd, zookeeper")
	flag.StringVar(&storeAddr, "addr", "", "The store cluster addr")
	flag.StringVar(&ip, "ip", "", "The ip of server lainlet running on")
	flag.BoolVar(&debug, "debug", false, "Open the Debug log")
	flag.BoolVar(&v, "v", false, "Print version")
	flag.BoolVar(&noAuth, "noauth", false, "whether close auth")
	flag.Parse()
}

func main() {
	if v {
		println("Lainlet:", version)
		return
	}
	if webAddr == "" || storeAddr == "" {
		flag.Usage()
		os.Exit(1)
	}
	if debug {
		log.EnableDebug()
	}

	s, err := store.New(storeType, strings.Split(storeAddr, ","))
	if err != nil {
		panic(err)
	}

	if err := auth.Init(s, context.Background(), ip, !noAuth); err != nil {
		panic(err)
	}

	server, err := api.New(ip, version, s)
	if err != nil {
		panic(err)
	}
	server.Register(new(v2.AppsData))
	server.Register(new(v2.GeneralConfig))
	server.Register(new(v2.GeneralPodGroup))
	server.Register(new(v2.GeneralCoreInfo))
	server.Register(new(v2.GeneralNodes))
	server.Register(new(v2.GeneralContainers))
	server.Register(new(v2.ProxyData))
	server.Register(new(v2.Depends))
	server.Register(new(v2.WebrouterInfo))
	server.Register(new(v2.RebellionAPIProvider))
	server.Register(new(v2.CoreInfoForBackupctl))
	server.Register(&v2.LocalSpec{
		LocalIP: ip,
	})
	server.Get("/appname", v2.GetAppNameAPI)

	go server.RunOnAddr(webAddr)

	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, os.Kill)
	<-ch

	// do something
}
