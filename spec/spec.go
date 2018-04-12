package spec

import (
	"time"

	"github.com/mijia/adoc"
)

type PodGroupWithSpec struct {
	Spec      PodGroupSpec
	PrevState []PodPrevState
	PodGroup
}

type PodGroupSpec struct {
	ImSpec
	Pod           PodSpec
	NumInstances  int
	RestartPolicy RestartPolicy
}

type ImSpec struct {
	Name      string
	Namespace string
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
}

type PodSpec struct {
	ImSpec
	Network      string
	Containers   []ContainerSpec
	Filters      []string // for cluster scheduling
	Labels       map[string]string
	Dependencies []Dependency
	Annotation   string
	Stateful     bool
	SetupTime    int
	KillTimeout  int
	PrevState    PodPrevState
	HealthConfig HealthConfig
}

type ContainerSpec struct {
	ImSpec
	Image         string
	Env           []string
	User          string
	WorkingDir    string
	DnsSearch     []string
	Volumes       []string // a stateful flag
	SystemVolumes []string // not a stateful flag, every node has system volumes
	CloudVolumes  []CloudVolumeSpec
	Command       []string
	Entrypoint    []string
	CpuLimit      int
	MemoryLimit   int64
	Expose        int
	LogConfig     adoc.LogConfig
}

type CloudVolumeSpec struct {
	Type string
	Dirs []string
}

type Dependency struct {
	PodName string
	Policy  DependencyPolicy
}

type DependencyPolicy int

type HealthConfig struct {
	Cmd     string           `json:"cmd"`
	Options HealthCnfOptions `json:"options"`
}

type HealthCnfOptions struct {
	Interval int `json:"interval"`
	Timeout  int `json:"timeout"`
	Retries  int `json:"retries"`
}

type RestartPolicy int

type PodPrevState struct {
	NodeName string
	IPs      []string
}

type PodGroup struct {
	Pods []Pod
	BaseRuntime
}

type Pod struct {
	InstanceNo int
	Containers []Container
	ImRuntime
}

type Container struct {
	// FIXME(mijia): multiple ports supporing, will have multiple entries of <NodePort, ContainerPort, Protocol>
	Id            string
	Runtime       adoc.ContainerDetail
	NodeName      string
	NodeIp        string
	ContainerIp   string
	NodePort      int
	ContainerPort int
	Protocol      string
}

type ImRuntime struct {
	BaseRuntime
	TargetState  ExpectState
	DriftCount   int
	RestartCount int
	RestartAt    time.Time
}

type BaseRuntime struct {
	Healthst  HealthState
	State     RunState
	OOMkilled bool
	LastError string
	UpdatedAt time.Time
}

type HealthState int

type RunState int

type ExpectState int

type SharedPodWithSpec struct {
	RefCount   int
	VerifyTime time.Time
	Spec       PodSpec
	Pod        Pod
}
