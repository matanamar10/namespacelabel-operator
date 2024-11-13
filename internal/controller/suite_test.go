package controller_test

import (
	"context"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	TestEnv   *envtest.Environment
	K8sClient client.Client
	Ctx       context.Context
	Scheme    = runtime.NewScheme()
	Cancel    context.CancelFunc
)

// Initialize the shared environment for tests
func Init() {
	Ctx, Cancel = context.WithCancel(context.TODO())

	TestEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
	}

	cfg, err := TestEnv.Start()
	if err != nil {
		panic(err)
	}

	utilruntime.Must(clientgoscheme.AddToScheme(Scheme))

	K8sClient, err = client.New(cfg, client.Options{Scheme: Scheme})
	if err != nil {
		panic(err)
	}
}

// Teardown cleans up the testing environment
func Teardown() {
	Cancel()
	if TestEnv != nil {
		if err := TestEnv.Stop(); err != nil {
			panic(err)
		}
	}
}
