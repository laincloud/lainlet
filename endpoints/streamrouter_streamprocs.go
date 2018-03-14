package endpoints

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/laincloud/lainlet/auth"
	pb "github.com/laincloud/lainlet/message"
	"github.com/laincloud/lainlet/store"
	"github.com/laincloud/lainlet/watcher"
	"github.com/laincloud/lainlet/watcher/podgroup"
)

type StreamrouterStreamprocsEndpoint struct {
	mu   sync.RWMutex
	name string
	wch  watcher.Watcher

	cache map[string]*pb.StreamrouterStreamprocsReply
}

func NewStreamrouterStreamprocsEndpoint(podgroupWatcher watcher.Watcher) *StreamrouterStreamprocsEndpoint {
	wh := &StreamrouterStreamprocsEndpoint{
		name:  "StreamrouterStreamprocs",
		wch:   podgroupWatcher,
		cache: make(map[string]*pb.StreamrouterStreamprocsReply),
	}
	return wh
}

func (ed *StreamrouterStreamprocsEndpoint) getKey(in *pb.StreamrouterStreamprocsRequest, ctx context.Context) (string, error) {
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

func (ed *StreamrouterStreamprocsEndpoint) make(key string, data map[string]interface{}) (*pb.StreamrouterStreamprocsReply, bool, error) {
	ret := &pb.StreamrouterStreamprocsReply{
		Data: make(map[string]*pb.StreamProcList),
	}
	var containerCount, aliveCount int
	for _, item := range data {
		pg := item.(podgroup.PodGroup)
		annotationStr := pg.Spec.Pod.Annotation
		var annotation pb.Annotation
		json.Unmarshal([]byte(annotationStr), &annotation)
		if len(annotation.Ports) == 0 {
			continue
		}
		proc := &pb.StreamProc{
			Name:      pg.Spec.Name,
			Upstreams: make([]*pb.StreamUpstream, len(pg.Pods)),
			Services:  make([]*pb.StreamService, len(annotation.Ports)),
		}

		for i, port := range annotation.Ports {
			proc.Services[i] = &pb.StreamService{
				UpstreamPort: port.Dstport,
				ListenPort:   port.Srcport,
			}
		}

		for i, pod := range pg.Pods {
			containerCount++
			proc.Upstreams[i] = &pb.StreamUpstream{
				Host:       pod.Containers[0].ContainerIp,
				InstanceNo: int32(pod.InstanceNo),
			}
			if len(pod.Containers) > 0 && len(pod.Containers[0].ContainerIp) > 0 {
				aliveCount++
			}
		}
		ret.Data[pg.Spec.Namespace].Procs = append(ret.Data[pg.Spec.Namespace].Procs, proc)
	}
	for _, v := range ret.Data {
		sort.Slice(v, func(i int, j int) bool { return v.Procs[i].Name < v.Procs[j].Name })
		for _, proc := range v.Procs {
			sort.Slice(proc.Upstreams, func(i int, j int) bool { return proc.Upstreams[i].InstanceNo < proc.Upstreams[j].InstanceNo })
			sort.Slice(proc.Services, func(i int, j int) bool { return proc.Services[i].ListenPort < proc.Services[j].ListenPort })
		}
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

//go:generate python ../tools/gen/srv.py 1 StreamrouterStreamprocs

//CODE GENERATION 1 START
func (ed *StreamrouterStreamprocsEndpoint) Get(ctx context.Context, in *pb.StreamrouterStreamprocsRequest) (*pb.StreamrouterStreamprocsReply, error) {
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

func (ed *StreamrouterStreamprocsEndpoint) Watch(in *pb.StreamrouterStreamprocsRequest, stream pb.StreamrouterStreamprocs_WatchServer) error {
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
