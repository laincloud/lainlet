package endpoints

import (
	"context"
	"errors"
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

var tooManyDeadContainersError = errors.New("over half of the containers lost their IPs")

type WebrouterWebprocsEndpoint struct {
	mu   sync.RWMutex
	name string
	wch  watcher.Watcher

	cache map[string]*pb.WebrouterWebprocsReply
}

func NewWebrouterWebprocsEndpoint(podgroupWatcher watcher.Watcher) *WebrouterWebprocsEndpoint {
	wh := &WebrouterWebprocsEndpoint{
		name:  "WebrouterWebprocs",
		wch:   podgroupWatcher,
		cache: make(map[string]*pb.WebrouterWebprocsReply),
	}
	return wh
}

func (ed *WebrouterWebprocsEndpoint) getKey(in *pb.WebrouterWebprocsRequest, ctx context.Context) (string, error) {
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

func (ed *WebrouterWebprocsEndpoint) make(key string, data map[string]interface{}) (*pb.WebrouterWebprocsReply, bool, error) {
	ret := &pb.WebrouterWebprocsReply{
		Data: make(map[string]*pb.CoreInfoForWebrouter),
	}
	var containerCount, aliveCount int
	for _, item := range data {
		pg := item.(podgroup.PodGroup)
		parts := strings.Split(pg.Spec.Name, ".")
		// Webrouter only cares about web procs
		if len(parts) < 3 || parts[len(parts)-2] != "web" {
			continue
		}
		ci := &pb.CoreInfoForWebrouter{
			PodInfos: make([]*pb.PodInfoForWebrouter, len(pg.Pods)),
		}
		for i, pod := range pg.Pods {
			containerCount++
			ci.PodInfos[i] = &pb.PodInfoForWebrouter{
				Annotation: pg.Spec.Pod.Annotation,
				Containers: make([]*pb.ContainerForWebrouter, len(pod.Containers)),
			}
			if (len(pod.Containers) > 0) && (len(pod.Containers[0].ContainerIp) > 0) {
				aliveCount++
			}
			for j, container := range pod.Containers {
				ci.PodInfos[i].Containers[j] = &pb.ContainerForWebrouter{
					IP:     container.ContainerIp,
					Expose: int32(pg.Spec.Pod.Containers[j].Expose),
				}
			}
		}
		ret.Data[pg.Spec.Name] = ci
	}
	if containerCount == 0 || aliveCount*2 < containerCount {
		return ret, false, tooManyDeadContainersError
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

//go:generate python ../tools/gen/srv.py 1 WebrouterWebprocs

//CODE GENERATION 1 START
func (ed *WebrouterWebprocsEndpoint) Get(ctx context.Context, in *pb.WebrouterWebprocsRequest) (*pb.WebrouterWebprocsReply, error) {
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

func (ed *WebrouterWebprocsEndpoint) Watch(in *pb.WebrouterWebprocsRequest, stream pb.WebrouterWebprocs_WatchServer) error {
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
