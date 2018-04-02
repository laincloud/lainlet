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

type AppsEndpoint struct {
	mu   sync.RWMutex
	name string
	wch  watcher.Watcher

	cache map[string]*pb.AppsReply
}

func NewAppsEndpoint(podgroup watcher.Watcher) *AppsEndpoint {
	wh := &AppsEndpoint{
		name:  "Apps",
		wch:   podgroup,
		cache: make(map[string]*pb.AppsReply),
	}
	return wh
}

func (ed *AppsEndpoint) getKey(in *pb.AppsRequest, ctx context.Context) (string, error) {
	remoteAddr, err := getRemoteAddr(ctx)
	if err != nil {
		return "", err
	}
	if !auth.IsSuper(remoteAddr) {
		return "", fmt.Errorf("authorize failed, super required")
	}
	return "*", nil
}

func (ed *AppsEndpoint) make(key string, data map[string]interface{}) (*pb.AppsReply, bool, error) {
	ret := &pb.AppsReply{
		Data: make(map[string]*pb.AppInfo),
	}
	for _, item := range data {
		pg := item.(podgroup.PodGroup)
		appname := pg.Spec.Namespace
		if _, ok := ret.Data[appname]; !ok {
			ret.Data[appname] = &pb.AppInfo{Appname: appname}
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

//go:generate python ../tools/gen/srv.py 1 Apps

//CODE GENERATION 1 START
func (ed *AppsEndpoint) Get(ctx context.Context, in *pb.AppsRequest) (*pb.AppsReply, error) {
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

func (ed *AppsEndpoint) Watch(in *pb.AppsRequest, stream pb.Apps_WatchServer) error {
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
