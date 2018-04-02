package endpoints

import (
	"context"
	"encoding/json"
	"runtime"

	pb "github.com/laincloud/lainlet/message"
	"github.com/laincloud/lainlet/version"
	"github.com/laincloud/lainlet/watcher"
)

type LainletEndpoint struct {
	Name     string
	Watchers map[string]watcher.Watcher
}

func NewLainletEndpoint(watchers map[string]watcher.Watcher) *LainletEndpoint {
	wh := &LainletEndpoint{
		Name:     "Lainlet",
		Watchers: watchers,
	}
	return wh
}

func (ed *LainletEndpoint) Version(ctx context.Context, in *pb.EmptyRequest) (*pb.VersionReply, error) {
	rpl := &pb.VersionReply{
		Version:    version.Version,
		APIVersion: version.APIVersion,
	}
	return rpl, nil
}

func (ed *LainletEndpoint) Status(ctx context.Context, in *pb.EmptyRequest) (*pb.StatusReply, error) {
	rpl := &pb.StatusReply{
		Goroutines: int32(runtime.NumGoroutine()),
		Status:     make(map[string]*pb.WatcherStatus),
	}
	for name, wch := range ed.Watchers {
		status := wch.Status()
		lastEvt, err := json.Marshal(status.LastEvent)
		if err != nil {
			return nil, err
		}
		pbStatus := &pb.WatcherStatus{
			NumReceivers: int32(status.NumReceivers),
			UpdateTime:   status.UpdateTime.Unix(),
			LastEvent:    lastEvt,
			TotalKeys:    int32(status.TotalKeys),
		}
		rpl.Status[name] = pbStatus
	}
	return rpl, nil
}
