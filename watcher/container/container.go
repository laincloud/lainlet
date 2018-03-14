package container

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/laincloud/deployd/engine"
	"github.com/laincloud/lainlet/store"
	"github.com/laincloud/lainlet/watcher"
	"github.com/mijia/sweb/log"
	"golang.org/x/net/context"
)

const (
	// KEY represents the key path in store
	KEY = "/lain/deployd/pod_groups"
)

// PodGroup is actually from the deployd engine, it actually engine.PodGroupWithSpec
type PodGroup engine.PodGroupWithSpec

// Info represents the container info, the data type returned by this container watcher
type Info struct {
	AppName    string `json:"app"`
	AppVersion string `json:"app_version"`
	ProcName   string `json:"proc"`
	NodeName   string `json:"nodename"`
	NodeIP     string `json:"nodeip"`
	IP         string `json:"ip"`
	Port       int    `json:"port"`
	InstanceNo int    `json:"instanceNo"`
}

type ContainerWatcher struct {
	watcher.BaseWatcher
	invertsTable map[string]string
}

// New create a new watcher which used to watch container info
func New(s store.Store, ctx context.Context) (watcher.Watcher, error) {
	ret := &ContainerWatcher{
		invertsTable:make(map[string]string),
	}
	base, err := watcher.New(s, ctx, KEY, ret.convert, ret.invertKey)
	if err != nil {
		return nil, err
	}
	ret.BaseWatcher = *base
	return ret, nil
}

func (wch *ContainerWatcher) invertKey(key string) string {
	s, ok := wch.invertsTable[key]
	if ok {
		return s
	}
	return ""
}

func (wch *ContainerWatcher) convert(pairs []*store.KVPair) (map[string]interface{}, error) {
	invertsTable := wch.invertsTable
	ret := make(map[string]interface{})
	for _, kv := range pairs {
		var pg PodGroup
		if err := json.Unmarshal(kv.Value, &pg); err != nil {
			log.Errorf("Fail to unmarshal the podgroup data: %s", string(kv.Value))
			log.Errorf("JSON unmarshal error: %s", err.Error())
			return nil, fmt.Errorf("a KVPair unmarshal failed")
		}

		/*
		 * we should delete old data in cache and invertTable.
		 * the etcd is set by deployd when app upgraded, if we do not delete here,
		 * the cache will having some containers which have been removed after upgrade
		 */
		deletedKeys := make([]string, 0, 2)
		for k, v := range invertsTable { // find the key should be deleted in invertsTable by the etcd key
			if v == kv.Key {
				deletedKeys = append(deletedKeys, k)
			}
		}
		for _, k := range deletedKeys {
			ret[k] = nil            // set to nil, let watcher delete it in cache
			delete(invertsTable, k) // delete it in invert table
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
			for _, container := range pod.Containers {
				k1 := fmt.Sprintf("%s/%s", container.NodeName, container.Id)
				k2 := fmt.Sprintf("%s/%s", container.NodeIp, container.Id)
				ci := Info{
					AppName:    pg.Spec.Namespace,
					AppVersion: appVersion,
					ProcName:   pg.Spec.Name,
					NodeName:   container.NodeName,
					NodeIP:     container.NodeIp,
					IP:         container.ContainerIp,
					Port:       container.ContainerPort,
					InstanceNo: pod.InstanceNo,
				}

				invertsTable[k1] = kv.Key // lock in case of concurrent ?
				invertsTable[k2] = kv.Key // lock in case of concurrent ?
				ret[k1], ret[k2] = ci, ci
			}
		}

	}
	return ret, nil
}
