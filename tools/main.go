package main

import (
	"fmt"

	"encoding/json"

	"github.com/laincloud/lainlet/grpcclient"
)

func main() {
	var data []byte
	cfg := &grpcclient.Config{
		Addr: "127.0.0.1:9001",
	}
	cli, err := grpcclient.New(cfg)
	if err != nil {
		panic(err)
	}
	apps, err := cli.AppsGet()

	if err != nil {
		panic(err)
	}
	fmt.Printf("apps: %v\n\n", apps)

	config, err := cli.ConfigGet("")
	if err != nil {
		panic(err)
	}
	fmt.Printf("cfg: %v\n\n", config)

	containers, err := cli.ContainersGet("")
	if err != nil {
		panic(err)
	}
	fmt.Printf("containers: %v\n\n", containers)

	nodes, err := cli.NodesGet("")

	if err != nil {
		panic(err)
	}
	data, _ = json.Marshal(nodes)
	fmt.Printf("nodes: %v\n\n", string(data))

	ver, err := cli.Version()
	if err != nil {
		panic(err)
	}
	data, _ = json.Marshal(ver)
	fmt.Printf("version: %v\n\n", string(data))
	// watcher
	wch, err := cli.ConfigWatch("grpc")
	if err != nil {
		panic(err)
	}
	for {
		cfg, _ := wch.Next()
		fmt.Printf("=============\nwatch cfg %s\n", cfg.String())
	}
}
