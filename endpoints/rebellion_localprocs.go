package endpoints

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/laincloud/lainlet/auth"
	pb "github.com/laincloud/lainlet/message"
	"github.com/laincloud/lainlet/store"
	"github.com/laincloud/lainlet/watcher"
	"github.com/laincloud/lainlet/watcher/podgroup"
)

type RebellionLocalprocsEndpoint struct {
	mu   sync.RWMutex
	name string
	wch  watcher.Watcher

	cache map[string]*pb.RebellionLocalprocsReply
}

func NewRebellionLocalprocsEndpoint(podgroupWatcher watcher.Watcher) *RebellionLocalprocsEndpoint {
	wh := &RebellionLocalprocsEndpoint{
		name:  "RebellionLocalprocs",
		wch:   podgroupWatcher,
		cache: make(map[string]*pb.RebellionLocalprocsReply),
	}
	return wh
}

func (ed *RebellionLocalprocsEndpoint) getKey(in *pb.RebellionLocalprocsRequest, ctx context.Context) (string, error) {
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

func (ed *RebellionLocalprocsEndpoint) make(key string, data map[string]interface{}) (*pb.RebellionLocalprocsReply, bool, error) {
	hostName, _ := os.Hostname()
	ret := &pb.RebellionLocalprocsReply{
		Data: make(map[string]*pb.CoreInfoForRebellion),
	}
	for _, item := range data {
		pg := item.(podgroup.PodGroup)
		ci := &pb.CoreInfoForRebellion{
			PodInfos: make([]*pb.PodInfoForRebellion, 0, len(pg.Pods)),
		}

		var appVersion string
		if len(pg.Spec.Pod.Containers) > 0 {
			for _, envStr := range pg.Spec.Pod.Containers[0].Env {
				envParts := strings.Split(envStr, "=")
				if len(envParts) == 2 && envParts[0] == "LAIN_APP_RELEASE_VERSION" {
					appVersion = envParts[1]
					break
				}
			}
		}

		for _, pod := range pg.Pods {
			isLocalContainer := false
			for _, container := range pod.Containers {
				if container.NodeName == hostName {
					isLocalContainer = true
					break
				}
			}
			if isLocalContainer {
				ci.PodInfos = append(
					ci.PodInfos,
					&pb.PodInfoForRebellion{
						Annotation: pg.Spec.Pod.Annotation,
						InstanceNo: int32(pod.InstanceNo),
						AppVersion: appVersion,
					})
			}
		}
		if len(ci.PodInfos) > 0 {
			ret.Data[pg.Spec.Name] = ci
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

//go:generate python ../tools/gen/srv.py 1 RebellionLocalprocs

//CODE GENERATION 1 START
func (ed *RebellionLocalprocsEndpoint) Get(ctx context.Context, in *pb.RebellionLocalprocsRequest) (*pb.RebellionLocalprocsReply, error) {
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

func (ed *RebellionLocalprocsEndpoint) Watch(in *pb.RebellionLocalprocsRequest, stream pb.RebellionLocalprocs_WatchServer) error {
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
