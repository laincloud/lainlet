package endpoints

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/laincloud/lainlet/auth"
	pb "github.com/laincloud/lainlet/message"
	"github.com/laincloud/lainlet/watcher"
	"github.com/laincloud/lainlet/watcher/container"
)

type LocalspecEndpoint struct {
	mu      sync.RWMutex
	name    string
	wch     watcher.Watcher
	localIp string

	cache map[string]*pb.LocalspecReply
}

func NewLocalspecEndpoint(containerWatcher watcher.Watcher, ip string) *LocalspecEndpoint {
	wh := &LocalspecEndpoint{
		name:  "CoreInfo",
		wch:   containerWatcher,
		cache: make(map[string]*pb.LocalspecReply),
	}
	return wh
}

func (ed *LocalspecEndpoint) make(key string, data map[string]interface{}) (*pb.LocalspecReply, bool) {
	ret := &pb.LocalspecReply{
		Data: make([]string, 0),
	}
	// merge the repeat item, so using map
	set := make(map[string]bool)
	for _, item := range data {
		ci := item.(container.Info)
		set[fmt.Sprintf("%s/%s", ci.AppName, ci.ProcName)] = true
	}
	for k, _ := range set {
		ret.Data = append(ret.Data, k)
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
	return ret, changed
}

func (ed *LocalspecEndpoint) getKey(in *pb.LocalspecRequest, ctx context.Context) (string, error) {
	remoteAddr, err := getRemoteAddr(ctx)
	if err != nil {
		return "", err
	}
	if !auth.IsSuper(remoteAddr) {
		return "", fmt.Errorf("authorize failed, super required")
	}
	nodeip := ed.localIp
	if in.Nodeip != "" {
		nodeip = in.Nodeip
	}
	nodeip = fixPrefix(nodeip)
	return nodeip, nil
}

func (ed *LocalspecEndpoint) Get(ctx context.Context, in *pb.LocalspecRequest) (*pb.LocalspecReply, error) {
	key, err := ed.getKey(in, ctx)
	if err != nil {
		return nil, err
	}
	data, err := ed.wch.Get(key)
	if err != nil {
		return nil, err
	}
	obj, _ := ed.make(key, data)
	return obj, nil
}
