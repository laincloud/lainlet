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

type StreamrouterPortsEndpoint struct {
	mu   sync.RWMutex
	name string
	wch  watcher.Watcher

	cache map[string]*pb.StreamrouterPortsReply
}

func NewStreamrouterPortsEndpoint(podgroupWatcher watcher.Watcher) *StreamrouterPortsEndpoint {
	wh := &StreamrouterPortsEndpoint{
		name:  "StreamrouterPorts",
		wch:   podgroupWatcher,
		cache: make(map[string]*pb.StreamrouterPortsReply),
	}
	return wh
}

func (ed *StreamrouterPortsEndpoint) getKey(in *pb.StreamrouterPortsRequest, ctx context.Context) (string, error) {
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

func (ed *StreamrouterPortsEndpoint) make(key string, data map[string]interface{}) (*pb.StreamrouterPortsReply, bool, error) {
	ret := &pb.StreamrouterPortsReply{}
	for _, item := range data {
		pg := item.(podgroup.PodGroup)
		annotationStr := pg.Spec.Pod.Annotation
		var annotation pb.Annotation
		json.Unmarshal([]byte(annotationStr), &annotation)
		if len(annotation.Ports) == 0 {
			continue
		}
		for _, port := range annotation.Ports {
			ret.Data = append(ret.Data, port.Srcport)
		}
	}
	sort.Slice(ret.Data, func(i int, j int) bool { return ret.Data[i] < ret.Data[j] })

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

//go:generate python ../tools/gen/srv.py 1 StreamrouterPorts

//CODE GENERATION 1 START
func (ed *StreamrouterPortsEndpoint) Get(ctx context.Context, in *pb.StreamrouterPortsRequest) (*pb.StreamrouterPortsReply, error) {
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

func (ed *StreamrouterPortsEndpoint) Watch(in *pb.StreamrouterPortsRequest, stream pb.StreamrouterPorts_WatchServer) error {
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
