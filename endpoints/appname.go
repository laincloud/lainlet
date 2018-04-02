package endpoints

import (
	"context"
	"fmt"

	"github.com/laincloud/lainlet/auth"
	pb "github.com/laincloud/lainlet/message"
)

type AppnameEndpoint struct {
	name string
}

func NewAppnameEndpoint() *AppnameEndpoint {
	wh := &AppnameEndpoint{
		name:  "Appname",
	}
	return wh
}

func (ed *AppnameEndpoint) Get(ctx context.Context, in *pb.AppnameRequest) (*pb.AppnameReply, error) {
	remoteAddr, err := getRemoteAddr(ctx)
	if err != nil {
		return nil, err
	}
	if !auth.IsSuper(remoteAddr) {
		return nil, fmt.Errorf("authorize failed, super required")
	}
	ip := remoteAddr
	if in.Ip != "" {
		ip = in.Ip
	}
	appname, err := auth.AppName(ip)
	if err != nil {
		return nil, err
	}
	ret := &pb.AppnameReply{
		Data:make(map[string]string),
	}
	ret.Data["appname"] = appname
	return ret, nil
}
