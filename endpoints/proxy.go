package endpoints

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/laincloud/lainlet/auth"
	pb "github.com/laincloud/lainlet/message"
	"github.com/laincloud/lainlet/store"
	"github.com/laincloud/lainlet/watcher"
	"github.com/laincloud/lainlet/watcher/podgroup"
)

type ProxyEndpoint struct {
	mu   sync.RWMutex
	name string
	wch  watcher.Watcher

	cache map[string]*pb.ProxyReply
}

func NewProxyEndpoint(podgroupWatcher watcher.Watcher) *ProxyEndpoint {
	wh := &ProxyEndpoint{
		name:  "Proxy",
		wch:   podgroupWatcher,
		cache: make(map[string]*pb.ProxyReply),
	}
	return wh
}

func (ed *ProxyEndpoint) make(key string, data map[string]interface{}) (*pb.ProxyReply, bool, error) {
	ret := &pb.ProxyReply{
		Data: make(map[string]*pb.ProcInfo),
	}
	for _, item := range data {
		pg := item.(podgroup.PodGroup)
		pi := &pb.ProcInfo{
			Containers: make([]*pb.ContainerForProxy, 0),
		}
		for _, pod := range pg.Pods {
			for j, container := range pod.Containers {
				pi.Containers = append(pi.Containers, &pb.ContainerForProxy{
					ContainerIp:   container.ContainerIp,
					ContainerPort: int32(pg.Spec.Pod.Containers[j].Expose),
				})
			}

		}
		ret.Data[pg.Spec.Name] = pi
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

func (ed *ProxyEndpoint) getKey(in *pb.ProxyRequest, ctx context.Context) (string, error) {
	remoteAddr, err := getRemoteAddr(ctx)
	if err != nil {
		return "", err
	}

	appName := "*"
	if in.Appname != "" {
		appName = in.Appname
	}
	if !auth.Pass(remoteAddr, appName) {
		if appName == "*" { // try to set the appname automatically by remoteip
			appName, err := auth.AppName(remoteAddr)
			if err != nil {
				return "", fmt.Errorf("authorize failed, can not confirm the app by request ip")
			}
			return fixPrefix(appName), nil
		}
		return "", fmt.Errorf("authorize failed, no permission")
	}
	if appName != "*" {
		appName = fixPrefix(appName)
	}
	return appName, nil
}

//go:generate python ../tools/gen/srv.py 1 Proxy

//CODE GENERATION 1 START
func (ed *ProxyEndpoint) Get(ctx context.Context, in *pb.ProxyRequest) (*pb.ProxyReply, error) {
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

func (ed *ProxyEndpoint) Watch(in *pb.ProxyRequest, stream pb.Proxy_WatchServer) error {
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
