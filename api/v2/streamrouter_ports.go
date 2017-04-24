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
	"sort"
)

type Ports struct {
	Data []int
}

func (si *Ports) Decode(r []byte) error {
	return json.Unmarshal(r, &si.Data)
}

func (si *Ports) Encode() ([]byte, error) {
	return json.Marshal(si.Data)
}

func (si *Ports) URI() string {
	return "/streamrouter/ports"
}

func (si *Ports) WatcherName() string {
	return watcher.PODGROUP
}

func (si *Ports) Make(data map[string]interface{}) (api.API, bool, error) {
	ret := &Ports{
		Data: []int{},
	}
	for _, item := range data {
		pg := item.(podgroup.PodGroup)
		annotationStr := pg.Spec.Pod.Annotation
		var annotation Annotation
		json.Unmarshal([]byte(annotationStr), &annotation)
		if len(annotation.Ports) == 0 {
			continue
		}
		for _, port := range annotation.Ports {
			ret.Data = append(ret.Data, port.Srcport)
		}
	}
	sort.Slice(ret.Data, func(i int, j int) bool { return ret.Data[i] < ret.Data[j] })
	return ret, !reflect.DeepEqual(si.Data, ret.Data), nil
}

func (si *Ports) Key(r *http.Request) (string, error) {
	appName := api.GetString(r, "appname", "*")
	if !auth.Pass(r.RemoteAddr, appName) {
		if appName == "*" { // try to set the appname automatically by remoteip
			appName, err := auth.AppName(r.RemoteAddr)
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
