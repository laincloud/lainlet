package endpoints

import (
	"context"
	"fmt"
	"sync"

	"github.com/laincloud/lainlet/auth"
	pb "github.com/laincloud/lainlet/message"
	"github.com/laincloud/lainlet/store"
	"github.com/laincloud/lainlet/watcher"
	"github.com/laincloud/lainlet/watcher/container"
)

type ContainersEndpoint struct {
	mu   sync.RWMutex
	name string
	wch  watcher.Watcher

	cache map[string]*pb.ContainersReply
}

func NewContainersEndpoint(containerWatcher watcher.Watcher) *ContainersEndpoint {
	wh := &ContainersEndpoint{
		name:  "Containers",
		wch:   containerWatcher,
		cache: make(map[string]*pb.ContainersReply),
	}
	return wh
}

func (ed *ContainersEndpoint) make(key string, data map[string]interface{}) (*pb.ContainersReply, bool, error) {
	ret := &pb.ContainersReply{
		Data: make(map[string]*pb.Info),
	}
	for k, v := range data {
		if nil == v {
			continue
		}
		cinfo := v.(container.Info)
		ct := &pb.Info{
			AppName:    cinfo.AppName,
			AppVersion: cinfo.AppVersion,
			ProcName:   cinfo.ProcName,
			NodeName:   cinfo.NodeName,
			NodeIP:     cinfo.NodeIP,
			IP:         cinfo.IP,
			Port:       int32(cinfo.Port),
			InstanceNo: int32(cinfo.InstanceNo),
		}
		ret.Data[k] = ct
	}
	return ret, true, nil
}

func (ed *ContainersEndpoint) getKey(in *pb.ContainersRequest, ctx context.Context) (string, error) {
	remoteAddr, err := getRemoteAddr(ctx)
	if err != nil {
		return "", err
	}
	if !auth.IsSuper(remoteAddr) {
		return "", fmt.Errorf("authorize failed, super required")
	}
	key := "*"
	if in.Nodename != "" {
		key = in.Nodename
	}
	if key != "*" {
		key = fixPrefix(key)
	}

	return key, nil
}

//go:generate python ../tools/gen/srv.py 1 Containers

//CODE GENERATION 1 START
func (ed *ContainersEndpoint) Get(ctx context.Context, in *pb.ContainersRequest) (*pb.ContainersReply, error) {
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

func (ed *ContainersEndpoint) Watch(in *pb.ContainersRequest, stream pb.Containers_WatchServer) error {
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
