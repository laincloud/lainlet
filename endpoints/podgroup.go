package endpoints

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/laincloud/lainlet/auth"
	pb "github.com/laincloud/lainlet/message"
	"github.com/laincloud/lainlet/store"
	"github.com/laincloud/lainlet/watcher"
	"github.com/laincloud/lainlet/watcher/podgroup"
)

type PodgroupEndpoint struct {
	mu   sync.RWMutex
	name string
	wch  watcher.Watcher

	cache map[string]*pb.PodgroupReply
}

func NewPodgroupEndpoint(podgroupWatcher watcher.Watcher) *PodgroupEndpoint {
	wh := &PodgroupEndpoint{
		name:  "Podgroup",
		wch:   podgroupWatcher,
		cache: make(map[string]*pb.PodgroupReply),
	}
	return wh
}

func (ed *PodgroupEndpoint) make(key string, data map[string]interface{}) (*pb.PodgroupReply, bool, error) {
	ret := &pb.PodgroupReply{
		Data: make([]*pb.PodGroup, 0, len(data)),
	}
	for _, procInfo := range data {
		pg := &pb.PodGroup{
			Pods: make([]*pb.Pod, 0),
		}
		for _, instanceInfo := range procInfo.(podgroup.PodGroup).Pods {
			envlist := instanceInfo.Containers[0].Runtime.Config.Env
			procname := ""
			for _, str := range envlist {
				if strings.HasPrefix(str, "LAIN_PROCNAME=") {
					procname = str[len("LAIN_PROCNAME="):]
				}
			}
			inst := &pb.Pod{
				ProcName:   procname,
				InstanceNo: int32(instanceInfo.InstanceNo),
				IP:         instanceInfo.Containers[0].ContainerIp,
				Port:       int32(instanceInfo.Containers[0].ContainerPort),
			}
			pg.Pods = append(pg.Pods, inst)
		}
		ret.Data = append(ret.Data, pg)
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

func (ed *PodgroupEndpoint) getKey(in *pb.PodgroupRequest, ctx context.Context) (string, error) {
	remoteAddr, err := getRemoteAddr(ctx)
	if err != nil {
		return "", err
	}

	appName := in.Appname
	if appName == "" {
		return "", fmt.Errorf("appname reequired")
	}
	if !auth.Pass(remoteAddr, appName) {
		return "", fmt.Errorf("authorize failed, no permission")
	}
	if appName != "*" {
		appName = fixPrefix(appName)
	}
	return appName, nil
}

//go:generate python ../tools/gen/srv.py 1 Podgroup

//CODE GENERATION 1 START
func (ed *PodgroupEndpoint) Get(ctx context.Context, in *pb.PodgroupRequest) (*pb.PodgroupReply, error) {
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

func (ed *PodgroupEndpoint) Watch(in *pb.PodgroupRequest, stream pb.Podgroup_WatchServer) error {
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
