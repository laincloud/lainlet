package main

import (
	"flag"
	"os"
	"runtime"
	"strings"

	"os/signal"

	"fmt"

	"github.com/laincloud/lainlet/api"
	"github.com/laincloud/lainlet/api/v2"
	"github.com/laincloud/lainlet/auth"
	grpcserver "github.com/laincloud/lainlet/server"
	"github.com/laincloud/lainlet/store"
	_ "github.com/laincloud/lainlet/store/etcd"
	"github.com/laincloud/lainlet/version"
	"github.com/laincloud/lainlet/watcher"
	"github.com/laincloud/lainlet/watcher/config"
	"github.com/laincloud/lainlet/watcher/container"
	"github.com/laincloud/lainlet/watcher/depends"
	"github.com/laincloud/lainlet/watcher/nodes"
	"github.com/laincloud/lainlet/watcher/podgroup"
	"github.com/mijia/sweb/log"
	"golang.org/x/net/context"
)

var (
	webAddr, etcdAddr, ip string
	debug, v, noAuth      bool

	grpcTls                   bool
	grpcKeyFile, grpcCertFile string
	grpcAddr                  string
)

func init() {
	flag.StringVar(&webAddr, "web", "", "The address lainlet listen")
	flag.StringVar(&etcdAddr, "etcd", "", "Etcd cluster entry point like http://127.0.0.1:4001")
	flag.StringVar(&ip, "ip", "", "The ip of server lainlet running on")
	flag.BoolVar(&debug, "debug", false, "Open the Debug log")
	flag.BoolVar(&v, "v", false, "Print version")
	flag.BoolVar(&noAuth, "noauth", false, "whether close auth")

	flag.StringVar(&grpcAddr, "grpc.addr", "", "grpc address")
	flag.BoolVar(&grpcTls, "grpc.TLS", false, "enable TLS")
	flag.StringVar(&grpcKeyFile, "grpc.key", "", "TLS key file")
	flag.StringVar(&grpcCertFile, "grpc.cert", "", "TLS certification file")
	flag.Parse()
}

func initWatchers(st store.Store) (map[string]watcher.Watcher, error) {
	background := context.Background()
	configWatcher, err := config.New(st, background)
	if err != nil {
		return nil, err
	}
	containerWatcher, err := container.New(st, background)
	if err != nil {
		return nil, err
	}
	podgroupWatcher, err := podgroup.New(st, background)
	if err != nil {
		return nil, err
	}
	dependsWatcher, err := depends.New(st, background)
	if err != nil {
		return nil, err
	}
	nodesWatcher, err := nodes.New(st, background)
	if err != nil {
		return nil, err
	}
	ret := map[string]watcher.Watcher{
		"configwatcher":    configWatcher,
		"containerwatcher": containerWatcher,
		"podgroupwatcher":  podgroupWatcher,
		"dependswatcher":   dependsWatcher,
		"nodeswatcher":     nodesWatcher,
	}
	return ret, nil
}

func main() {
	if v {
		println("Lainlet Version:", version.Version)
		fmt.Printf("Git SHA: %s\n", version.GitSHA)
		fmt.Printf("Go Version: %s\n", runtime.Version())
		return
	}
	if webAddr == "" && grpcAddr == "" {
		fmt.Println("you should at least specify one of webAddr and grpcAddr.")
		os.Exit(-1)
	}
	if etcdAddr == "" {
		flag.Usage()
		os.Exit(1)
	}
	if debug {
		log.EnableDebug()
	}

	st, err := store.New("etcd", strings.Split(etcdAddr, ","))
	if err != nil {
		panic(err)
	}

	if err := auth.Init(st, context.Background(), ip, !noAuth); err != nil {
		panic(err)
	}
	watchers, err := initWatchers(st)
	if err != nil {
		panic(err)
	}

	if webAddr != "" {
		httpSrv, err := api.New(ip, version.Version, watchers)
		if err != nil {
			panic(err)
		}
		httpSrv.Register(new(v2.AppsData))
		httpSrv.Register(new(v2.GeneralConfig))
		httpSrv.Register(new(v2.GeneralPodGroup))
		httpSrv.Register(new(v2.GeneralCoreInfo))
		httpSrv.Register(new(v2.GeneralNodes))
		httpSrv.Register(new(v2.GeneralContainers))
		httpSrv.Register(new(v2.ProxyData))
		httpSrv.Register(new(v2.Depends))
		httpSrv.Register(new(v2.WebrouterInfo))
		httpSrv.Register(new(v2.StreamRouterInfo))
		httpSrv.Register(new(v2.Ports))
		httpSrv.Register(new(v2.RebellionAPIProvider))
		httpSrv.Register(new(v2.CoreInfoForBackupctl))
		httpSrv.Register(&v2.LocalSpec{
			LocalIP: ip,
		})
		httpSrv.Get("/appname", v2.GetAppNameAPI)

		go httpSrv.RunOnAddr(webAddr)
	}

	if grpcAddr != "" {
		var cfg *grpcserver.Config
		if grpcTls {
			cfg = grpcserver.NewConfig(grpcAddr, grpcKeyFile, grpcCertFile)
		}
		grpcSrv, err := grpcserver.New(grpcAddr, ip, watchers, cfg)
		if err != nil {
			panic(err)
		}
		go grpcSrv.Run()
	}

	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, os.Kill)
	<-ch
}
