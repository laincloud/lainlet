package endpoints

import (
	"context"
	"fmt"
	"sync"

	"reflect"

	"github.com/laincloud/lainlet/auth"
	pb "github.com/laincloud/lainlet/message"
	"github.com/laincloud/lainlet/store"
	"github.com/laincloud/lainlet/watcher"
	"github.com/laincloud/lainlet/watcher/depends"
)

type DependsEndpoint struct {
	mu   sync.RWMutex
	name string
	wch  watcher.Watcher

	cache map[string]*pb.DependsReply
}

func NewDependsEndpoint(dependsWatcher watcher.Watcher) *DependsEndpoint {
	wh := &DependsEndpoint{
		name:  "Depends",
		wch:   dependsWatcher,
		cache: make(map[string]*pb.DependsReply),
	}
	return wh
}

func (ed *DependsEndpoint) make(key string, data map[string]interface{}) (*pb.DependsReply, bool, error) {
	ret := &pb.DependsReply{
		Data: make(map[string]*pb.DependsNodeMap),
	}
	for key, item := range data {
		dd := item.(depends.Depends)
		ret.Data[key] = &pb.DependsNodeMap{
			Nodes: make(map[string]*pb.DependsAppMap),
		}
		for nodeName, appList := range dd {
			depApp := &pb.DependsAppMap{
				Apps: make(map[string]*pb.DependsItem),
			}
			ret.Data[key].Nodes[nodeName] = depApp
			for app, item := range appList {
				containers := make([]*pb.ContainerInfo, 0, len(item.Pod.Containers))
				for i, _ := range item.Pod.Containers {
					containers = append(containers, &pb.ContainerInfo{
						ContainerID: item.Pod.Containers[i].Id,
						NodeIP:      item.Pod.Containers[i].NodeIp,
						IP:          item.Pod.Containers[i].ContainerIp,
						Port:        int32(item.Pod.Containers[i].ContainerPort),
					})
				}
				depApp.Apps[app] = &pb.DependsItem{
					Annotation: item.Spec.Annotation,
					Containers: containers,
				}
			}
		}
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

func (ed *DependsEndpoint) getKey(in *pb.DependsRequest, ctx context.Context) (string, error) {
	remoteAddr, err := getRemoteAddr(ctx)
	if err != nil {
		return "", err
	}
	if !auth.IsSuper(remoteAddr) {
		return "", fmt.Errorf("authorize failed, super required")
	}
	key := "*"
	if in.Target != "" {
		key = in.Target
	}
	return key, nil
}

//go:generate python ../tools/gen/srv.py 1 Depends

//CODE GENERATION 1 START
func (ed *DependsEndpoint) Get(ctx context.Context, in *pb.DependsRequest) (*pb.DependsReply, error) {
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

func (ed *DependsEndpoint) Watch(in *pb.DependsRequest, stream pb.Depends_WatchServer) error {
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
