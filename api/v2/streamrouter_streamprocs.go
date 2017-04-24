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

type StreamProc struct {
	Name      string
	Upstreams []StreamUpstream
	Services  []StreamService
}

type StreamUpstream struct {
	Host       string
	InstanceNo int
}

type StreamService struct {
	UpstreamPort int
	ListenPort   int
	Send         string
	Expect       string
}

type StreamRouterInfo struct {
	Data map[string][]StreamProc
}

type Annotation struct {
	Ports []Port `json:"ports"`
}

type Port struct {
	Srcport     int         `json:"srcport"`
	Dstport     int         `json:"dstport"`
	Proto       string      `json:"proto"`
}

func (si *StreamRouterInfo) Decode(r []byte) error {
	return json.Unmarshal(r, &si.Data)
}

func (si *StreamRouterInfo) Encode() ([]byte, error) {
	return json.Marshal(si.Data)
}

func (si *StreamRouterInfo) URI() string {
	return "/streamrouter/streamprocs"
}

func (si *StreamRouterInfo) WatcherName() string {
	return watcher.PODGROUP
}

func (si *StreamRouterInfo) Make(data map[string]interface{}) (api.API, bool, error) {
	ret := &StreamRouterInfo{
		Data: make(map[string][]StreamProc),
	}
	var containerCount, aliveCount int
	for _, item := range data {
		pg := item.(podgroup.PodGroup)
		annotationStr := pg.Spec.Pod.Annotation
		var annotation Annotation
		json.Unmarshal([]byte(annotationStr), &annotation)
		if len(annotation.Ports) == 0 {
			continue
		}
		proc := StreamProc{
			Name:      pg.Spec.Name,
			Upstreams: make([]StreamUpstream, len(pg.Pods)),
			Services:  make([]StreamService, len(annotation.Ports)),
		}

		for i, port := range annotation.Ports {
			proc.Services[i] = StreamService{
				UpstreamPort: port.Dstport,
				ListenPort:   port.Srcport,
			}
		}

		for i, pod := range pg.Pods {
			containerCount++
			proc.Upstreams[i] = StreamUpstream{
				Host:       pod.Containers[0].ContainerIp,
				InstanceNo: pod.InstanceNo,
			}
			if len(pod.Containers) > 0 && len(pod.Containers[0].ContainerIp) > 0 {
				aliveCount++
			}
		}
		ret.Data[pg.Spec.Namespace] = append(ret.Data[pg.Spec.Namespace], proc)
	}
	for _, v := range ret.Data {
		sort.Slice(v, func(i int, j int) bool { return v[i].Name < v[j].Name })
		for _, proc := range v {
			sort.Slice(proc.Upstreams, func(i int, j int) bool { return proc.Upstreams[i].InstanceNo < proc.Upstreams[j].InstanceNo })
			sort.Slice(proc.Services, func(i int, j int) bool { return proc.Services[i].ListenPort < proc.Services[j].ListenPort })
		}
	}
	if containerCount == 0 || aliveCount*2 < containerCount {
		return ret, false, tooManyDeadContainersError
	}
	return ret, !reflect.DeepEqual(si.Data, ret.Data), nil
}

func (si *StreamRouterInfo) Key(r *http.Request) (string, error) {
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
