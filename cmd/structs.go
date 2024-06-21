package cmd

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	restapi "sigs.k8s.io/controller-runtime/pkg/client"
)

type KubernetesAPI struct {
	Rest    restapi.Client
	Client  kubernetes.Interface
	Dynamic dynamic.DynamicClient
	Config  *rest.Config
}

type ObjectStorage struct {
	AccessKey  string
	SecretKey  string
	Region     string
	Endpoint   string
	Type       string
	BucketName string
	Namespace  string
}

type Flags struct {
	Repo           string
	ChartName      string
	ReleaseName    string
	Values         string
	Domain         string
	DryRun         bool
	TargetRevision string
	ExternalIps    string
	App            bool
	Argocd         string
}
