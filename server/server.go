package server

import (
	"fmt"
	"net"

	"github.com/laincloud/lainlet/endpoints"
	pb "github.com/laincloud/lainlet/message"
	"github.com/laincloud/lainlet/watcher"
	"github.com/mijia/sweb/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Config struct {
	addr string

	keyFile  string
	certFile string
}

type Server struct {
	addr    string
	localIp string
	cfg     *Config

	watchers map[string]watcher.Watcher

	configWatcher    watcher.Watcher
	containerWatcher watcher.Watcher
	podgroupWatcher  watcher.Watcher
	dependsWatcher   watcher.Watcher
	nodesWatcher     watcher.Watcher
}

func NewConfig(addr, key, cert string) *Config {
	return &Config{
		addr:     addr,
		keyFile:  key,
		certFile: cert,
	}
}

func New(addr string, ip string, watchers map[string]watcher.Watcher, cfg *Config) (*Server, error) {
	if cfg != nil {
		if cfg.keyFile == "" || cfg.certFile == "" {
			return nil, fmt.Errorf("keyfile or certfile can't be empty when TLS is enabled.")
		}
	}
	srv := &Server{
		addr:    addr,
		localIp: ip,
		cfg:     cfg,

		watchers: watchers,

		configWatcher:    watchers["configwatcher"],
		containerWatcher: watchers["containerwatcher"],
		podgroupWatcher:  watchers["podgroupwatcher"],
		dependsWatcher:   watchers["dependswatcher"],
		nodesWatcher:     watchers["nodeswatcher"],
	}
	return srv, nil
}

func (srv *Server) Run() {
	lis, err := net.Listen("tcp", srv.addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	if srv.cfg != nil {
		creds, err := credentials.NewServerTLSFromFile(srv.cfg.certFile, srv.cfg.keyFile)
		if err != nil {
			log.Fatalf("Failed to generate credentials %v", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}
	grpcServer := grpc.NewServer(opts...)

	pb.RegisterAppnameServer(grpcServer, endpoints.NewAppnameEndpoint())
	pb.RegisterLainletServer(grpcServer, endpoints.NewLainletEndpoint(srv.watchers))

	pb.RegisterAppsServer(grpcServer, endpoints.NewAppsEndpoint(srv.podgroupWatcher))
	pb.RegisterBackupctlServer(grpcServer, endpoints.NewBackupctlEndpoint(srv.podgroupWatcher))
	pb.RegisterConfigServer(grpcServer, endpoints.NewConfigEndpoint(srv.configWatcher))
	pb.RegisterContainersServer(grpcServer, endpoints.NewContainersEndpoint(srv.containerWatcher))
	pb.RegisterCoreinfoServer(grpcServer, endpoints.NewCoreinfoEndpoint(srv.podgroupWatcher))
	pb.RegisterDependsServer(grpcServer, endpoints.NewDependsEndpoint(srv.dependsWatcher))
	pb.RegisterLocalspecServer(grpcServer, endpoints.NewLocalspecEndpoint(srv.containerWatcher, srv.localIp))
	pb.RegisterNodesServer(grpcServer, endpoints.NewNodesEndpoint(srv.nodesWatcher))
	pb.RegisterPodgroupServer(grpcServer, endpoints.NewPodgroupEndpoint(srv.podgroupWatcher))
	pb.RegisterProxyServer(grpcServer, endpoints.NewProxyEndpoint(srv.podgroupWatcher))
	pb.RegisterRebellionLocalprocsServer(grpcServer, endpoints.NewRebellionLocalprocsEndpoint(srv.podgroupWatcher))
	pb.RegisterStreamrouterPortsServer(grpcServer, endpoints.NewStreamrouterPortsEndpoint(srv.podgroupWatcher))
	pb.RegisterStreamrouterStreamprocsServer(grpcServer, endpoints.NewStreamrouterStreamprocsEndpoint(srv.podgroupWatcher))
	pb.RegisterWebrouterWebprocsServer(grpcServer, endpoints.NewWebrouterWebprocsEndpoint(srv.podgroupWatcher))

	grpcServer.Serve(lis)
}
