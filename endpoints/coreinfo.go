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

type CoreinfoEndpoint struct {
	mu   sync.RWMutex
	name string
	wch  watcher.Watcher

	cache map[string]*pb.CoreinfoReply
}

func NewCoreinfoEndpoint(podgroupWatcher watcher.Watcher) *CoreinfoEndpoint {
	wh := &CoreinfoEndpoint{
		name:  "CoreInfo",
		wch:   podgroupWatcher,
		cache: make(map[string]*pb.CoreinfoReply),
	}
	return wh
}

func (ed *CoreinfoEndpoint) make(key string, data map[string]interface{}) (*pb.CoreinfoReply, bool, error) {
	ret := &pb.CoreinfoReply{
		Data: make(map[string]*pb.CoreInfo),
	}
	for _, item := range data {
		pg := item.(podgroup.PodGroup)
		ci := &pb.CoreInfo{
			PodInfos: make([]*pb.PodInfo, len(pg.Pods)),
		}
		for i, pod := range pg.Pods {
			ci.PodInfos[i] = &pb.PodInfo{
				Annotation:   pg.Spec.Pod.Annotation,
				InstanceNo:   int32(pod.InstanceNo),
				Containers:   make([]*pb.Container, len(pod.Containers)),
				Dependencies: make([]*pb.Dependency, len(pg.Spec.Pod.Dependencies)),
			}
			for j, container := range pod.Containers {
				ci.PodInfos[i].Containers[j] = &pb.Container{
					Command:  pg.Spec.Pod.Containers[j].Command,
					Id:       container.Id,
					Ip:       container.ContainerIp,
					Cpu:      int32(pg.Spec.Pod.Containers[j].CpuLimit),
					Env:      pg.Spec.Pod.Containers[j].Env,
					Expose:   int32(pg.Spec.Pod.Containers[j].Expose),
					Image:    pg.Spec.Pod.Containers[j].Image,
					Memory:   pg.Spec.Pod.Containers[j].MemoryLimit,
					NodeIp:   container.NodeIp,
					NodeName: container.NodeName,
					Volumes:  pg.Spec.Pod.Containers[j].Volumes,
				}
			}
			for k, depend := range pg.Spec.Pod.Dependencies {
				ci.PodInfos[i].Dependencies[k] = &pb.Dependency{
					PodName: depend.PodName,
					Policy:  int32(depend.Policy),
				}
			}
		}
		ret.Data[pg.Spec.Name] = ci
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

func (ed *CoreinfoEndpoint) getKey(in *pb.CoreinfoRequest, ctx context.Context) (string, error) {
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

//go:generate python ../tools/gen/srv.py 1 Coreinfo

//CODE GENERATION 1 START
func (ed *CoreinfoEndpoint) Get(ctx context.Context, in *pb.CoreinfoRequest) (*pb.CoreinfoReply, error) {
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

func (ed *CoreinfoEndpoint) Watch(in *pb.CoreinfoRequest, stream pb.Coreinfo_WatchServer) error {
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
