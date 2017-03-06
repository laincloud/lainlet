package v2

import (
	"encoding/json"
	"fmt"
	"github.com/laincloud/lainlet/api"
	"github.com/laincloud/lainlet/auth"
	"github.com/laincloud/lainlet/watcher"
	"github.com/laincloud/lainlet/watcher/podgroup"
	"net/http"
	"reflect"
)

// Container info, aim to make it compatible with old api, having to defined a new struct.
type AppInfo struct {
	Appname string
}

type AppsData struct {
	Data map[string]AppInfo
}

func (ad *AppsData) Decode(r []byte) error {
	return json.Unmarshal(r, &ad.Data)
}

func (ad *AppsData) Encode() ([]byte, error) {
	return json.Marshal(ad.Data)
}

func (ad *AppsData) URI() string {
	return "/appswatcher"
}

func (ad *AppsData) WatcherName() string {
	return watcher.PODGROUP
}

func (ad *AppsData) Make(data map[string]interface{}) (api.API, bool, error) {
	ret := &AppsData{
		Data: make(map[string]AppInfo),
	}
	for _, item := range data {
		pg := item.(podgroup.PodGroup)
		appname := pg.Spec.Namespace
		if _, ok := ret.Data[appname]; !ok {
			ret.Data[appname] = AppInfo{Appname: appname}
		}
	}
	return ret, !reflect.DeepEqual(ad.Data, ret.Data), nil
}

func (ad *AppsData) Key(r *http.Request) (string, error) {
	if !auth.IsSuper(r.RemoteAddr) {
		return "", fmt.Errorf("authorize failed, super required")
	}
	return "*", nil
}
