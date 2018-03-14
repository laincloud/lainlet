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
	"github.com/laincloud/lainlet/watcher/nodes"
)

type NodesEndpoint struct {
	mu   sync.RWMutex
	name string
	wch  watcher.Watcher

	cache map[string]*pb.NodesReply
}

func NewNodesEndpoint(nodesWatcher watcher.Watcher) *NodesEndpoint {
	wh := &NodesEndpoint{
		name:  "Nodes",
		wch:   nodesWatcher,
		cache: make(map[string]*pb.NodesReply),
	}
	return wh
}

func (ed *NodesEndpoint) getKey(in *pb.NodesRequest, ctx context.Context) (string, error) {
	remoteAddr, err := getRemoteAddr(ctx)
	if err != nil {
		return "", err
	}
	if !auth.IsSuper(remoteAddr) {
		return "", fmt.Errorf("authorize failed, super required")
	}
	key := "*"
	if in.Name != "" {
		key = in.Name
	}
	if key != "*" && key[len(key)-1] != ':' {
		key += ":"
	}
	return key, nil
}

func (ed *NodesEndpoint) make(key string, data map[string]interface{}) (*pb.NodesReply, bool, error) {
	ret := &pb.NodesReply{
		Data: make(map[string]*pb.NodeInfo),
	}
	for k, v := range data {
		ni := v.(nodes.NodeInfo)
		pbNi := &pb.NodeInfo{
			V: make(map[string]*pb.NodeInfo_Value),
		}

		for niK, niV := range ni {
			var pbV *pb.NodeInfo_Value
			if sval, ok := niV.(string); ok {
				pbV = &pb.NodeInfo_Value{
					Vtype: pb.NodeInfo_Value_STRING,
					Sval:  sval,
				}
			} else if mval, ok := niV.(map[string]string); ok {
				pbV = &pb.NodeInfo_Value{
					Vtype: pb.NodeInfo_Value_MAP,
					Mval:  mval,
				}
			}
			pbNi.V[niK] = pbV
		}
		ret.Data[k] = pbNi
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

//go:generate python ../tools/gen/srv.py 1 Nodes

//CODE GENERATION 1 START
func (ed *NodesEndpoint) Get(ctx context.Context, in *pb.NodesRequest) (*pb.NodesReply, error) {
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

func (ed *NodesEndpoint) Watch(in *pb.NodesRequest, stream pb.Nodes_WatchServer) error {
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
