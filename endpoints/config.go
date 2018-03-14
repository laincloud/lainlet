package endpoints

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"reflect"

	"github.com/laincloud/lainlet/auth"
	pb "github.com/laincloud/lainlet/message"
	"github.com/laincloud/lainlet/store"
	"github.com/laincloud/lainlet/watcher"
)

var (
	secretKeys = []string{
		"*",
		"swarm_manager_ip",
		"super_apps",
		"dnsmasq_servers",
		"calico_default_rule",
		"calico_network",
		"dnsmasq_addresses",
		"ssl",
		"vips",
		"tinydns_fqdns",
		"bootstrap_node_ip",
		"dns_port",
		"vip",
		"etcd_cluster_token",
		"system_volumes",
		"rsyncd_secrets",
		"dns_ip",
		"node_network",
	}
)

func isSecret(key string) bool {
	for _, sk := range secretKeys {
		if strings.HasPrefix(key, sk) {
			return true
		}
	}
	return false
}

type ConfigEndpoint struct {
	mu   sync.RWMutex
	name string
	wch  watcher.Watcher

	cache map[string]*pb.ConfigReply
}

func NewConfigEndpoint(configWatcher watcher.Watcher) *ConfigEndpoint {
	wh := &ConfigEndpoint{
		name:  "Config",
		wch:   configWatcher,
		cache: make(map[string]*pb.ConfigReply),
	}
	return wh
}

func (ed *ConfigEndpoint) getKey(in *pb.ConfigRequest, ctx context.Context) (string, error) {
	remoteAddr, err := getRemoteAddr(ctx)
	if err != nil {
		return "", err
	}
	target := "*"
	if in.Target != "" {
		target = in.Target
	}
	if isSecret(target) && !auth.IsSuper(remoteAddr) {
		return "", fmt.Errorf("authorize failed, super required")
	}
	return target, nil
}

func (ed *ConfigEndpoint) make(key string, conf map[string]interface{}) (*pb.ConfigReply, bool, error) {
	ret := &pb.ConfigReply{
		Data: make(map[string]string),
	}
	for k, v := range conf {
		ret.Data[k], _ = v.(string)
	}
	changed := true
	ed.mu.Lock()
	ed.mu.Unlock()
	cached, _ := ed.cache[key]
	if cached != nil && reflect.DeepEqual(cached, ret) {
		changed = false
	} else {
		ed.cache[key] = ret
	}
	return ret, changed, nil
}

//go:generate python ../tools/gen/srv.py 1 Config

//CODE GENERATION 1 START
func (ed *ConfigEndpoint) Get(ctx context.Context, in *pb.ConfigRequest) (*pb.ConfigReply, error) {
	key, err := ed.getKey(in, ctx)
	if err != nil {
		return nil, err
	}
	data, err := ed.wch.Get(key)
	if err != nil {
		return nil, err
	}
	obj, _, err := ed.make(key, data)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (ed *ConfigEndpoint) Watch(in *pb.ConfigRequest, stream pb.Config_WatchServer) error {
 	ctx := stream.Context()
	key, err := ed.getKey(in, ctx)
	if err != nil {
		return err
	}

	// send the initial data
	data, err := ed.wch.Get(key)
	if err != nil {
		return err
	}
	obj, _, err := ed.make(key, data)
	if err != nil {
		return err
	}
	if err := stream.Send(obj); err != nil {
		return err
	}

	// start watching
	ch, err := ed.wch.Watch(key, ctx)
	if err != nil {
		return fmt.Errorf("Fail to watch %s, %s", key, err.Error())
	}
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return nil
			}
			// log.Infof("Get a %s event, id=%d action=%s ", api.WatcherName(), event.ID, event.Action.String())
			if event.Action == store.ERROR {
				err := fmt.Errorf("got an error from store, ID: %v, Data: %v", event.ID, event.Data)
				return err
			}
			obj, changed, err := ed.make(key, event.Data)
			if err != nil {
				return err
			}
			if !changed {
				continue
			}
			if err := stream.Send(obj); err != nil {
				return err
			}
		case  <-ctx.Done():
		    return nil
		}
	}
}

//CODE GENERATION 1 END
