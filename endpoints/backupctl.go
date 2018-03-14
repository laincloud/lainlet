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

type BackupctlEndpoint struct {
	mu   sync.RWMutex
	name string
	wch  watcher.Watcher

	cache map[string]*pb.BackupctlReply
}

func NewBackupctlEndpoint(podgroup watcher.Watcher) *BackupctlEndpoint {
	wh := &BackupctlEndpoint{
		name:  "Backupctl",
		wch:   podgroup,
		cache: make(map[string]*pb.BackupctlReply),
	}
	return wh
}

func (ed *BackupctlEndpoint) getKey(in *pb.BackupctlRequest, ctx context.Context) (string, error) {
	remoteAddr, err := getRemoteAddr(ctx)
	if err != nil {
		return "", err
	}
	if !auth.IsSuper(remoteAddr) {
		return "", fmt.Errorf("authorize failed, super required")
	}
	appname := "*"
	if in.Appname != "" {
		appname = in.Appname
	}
	if appname != "*" {
		appname = fixPrefix(appname)
	}
	return appname, nil
}

func (ed *BackupctlEndpoint) make(key string, data map[string]interface{}) (*pb.BackupctlReply, bool, error) {
	ret := &pb.BackupctlReply{
		Data: make(map[string]*pb.BackupctlReply_PodInfoList),
	}
	for _, item := range data {
		pg := item.(podgroup.PodGroup)
		infos := make([]*pb.PodInfoForBackupctl, len(pg.Pods))
		for i, pod := range pg.Pods {
			infos[i] = &pb.PodInfoForBackupctl{
				Annotation: pg.Spec.Pod.Annotation,
				InstanceNo: int32(pod.InstanceNo),
				Containers: make([]*pb.ContainerForBackupctl, len(pod.Containers)),
			}
			for j, container := range pod.Containers {
				infos[i].Containers[j] = &pb.ContainerForBackupctl{
					Id:       container.Id,
					Ip:       container.ContainerIp,
					NodeIp:   container.NodeIp,
					NodeName: container.NodeName,
				}
			}
		}
		ret.Data[pg.Spec.Name] = &pb.BackupctlReply_PodInfoList{Pods: infos}
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

//go:generate python ../tools/gen/srv.py 1 Backupctl

//CODE GENERATION 1 START
func (ed *BackupctlEndpoint) Get(ctx context.Context, in *pb.BackupctlRequest) (*pb.BackupctlReply, error) {
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

func (ed *BackupctlEndpoint) Watch(in *pb.BackupctlRequest, stream pb.Backupctl_WatchServer) error {
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
