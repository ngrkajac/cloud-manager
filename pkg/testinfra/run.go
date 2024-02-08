package testinfra

import (
	"errors"
	"fmt"
	awsmock "github.com/kyma-project/cloud-manager/pkg/kcp/provider/aws/mock"
	skrruntime "github.com/kyma-project/cloud-manager/pkg/skr/runtime"
	"github.com/onsi/ginkgo/v2"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	goruntime "runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func Start() (Infra, error) {
	projectRoot := os.Getenv("PROJECTROOT")
	if len(projectRoot) == 0 {
		return nil, errors.New("the env var PROJECTROOT must be specified and point to the dir where Makefile is")
	}
	envtestK8sVersion := os.Getenv("ENVTEST_K8S_VERSION")
	if len(envtestK8sVersion) == 0 {
		envtestK8sVersion = "1.28.0"
	}

	ginkgo.By("Preparing CRDs")

	dirSkr, dirKcp, dirGarden, err := initCrds(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("error initializing CRDs: %w", err)
	}

	configDir := filepath.Join(projectRoot, "bin", "cloud-manager", "config")
	if err := os.MkdirAll(configDir, 0777); err != nil {
		return nil, fmt.Errorf("error creating config dir: %w", err)
	}

	infra := &infra{
		clusters: map[ClusterType]*clusterInfo{
			ClusterTypeKcp: &clusterInfo{
				crdDirs: []string{dirKcp},
			},
			ClusterTypeSkr: &clusterInfo{
				crdDirs: []string{dirSkr},
			},
			ClusterTypeGarden: &clusterInfo{
				crdDirs: []string{dirGarden},
			},
		},
	}

	for name, cluster := range infra.clusters {
		ginkgo.By(fmt.Sprintf("Startig cluster %s", name))
		sch, ok := schemeMap[name]
		if !ok {
			return nil, fmt.Errorf("missing scheme for cluster %s", name)
		}

		env, cfg, err := startCluster(cluster.crdDirs, projectRoot, envtestK8sVersion)
		if err != nil {
			return nil, fmt.Errorf("error starting cluster %s: %w", name, err)
		}

		k8sClient, err := ctrlclient.New(cfg, ctrlclient.Options{Scheme: sch})
		if err != nil {
			return nil, fmt.Errorf("error creating client for %s: %w", name, err)
		}

		cluster.env = env
		cluster.cfg = cfg
		cluster.scheme = sch
		cluster.client = k8sClient

		ce := &clusterEnv{}
		switch name {
		case ClusterTypeKcp:
			ce.namespace = "kcp-system"
		case ClusterTypeSkr:
			ce.namespace = "kyma-system"
		case ClusterTypeGarden:
			ce.namespace = "garden-kyma"
		}
		cluster.ClusterEnv = ce
	}

	ginkgo.By("All started")

	// Create ENV
	kcpMgr, err := ctrl.NewManager(infra.clusters[ClusterTypeKcp].cfg, ctrl.Options{
		Scheme: infra.KCP().Scheme(),
		Client: ctrlclient.Options{
			Cache: &ctrlclient.CacheOptions{
				Unstructured: true,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error creating KCP manager: %w", err)
	}

	registry := skrruntime.NewRegistry(infra.SKR().Scheme())
	looper := skrruntime.NewLooper(kcpMgr, infra.SKR().Scheme(), registry, kcpMgr.GetLogger())

	awsMock := awsmock.New()
	awsMock.SetAccount("some-aws-account")

	infra.InfraEnv = &infraEnv{
		i:          infra,
		kcpManager: kcpMgr,
		registry:   registry,
		looper:     looper,
		awsMock:    awsMock,
		skrKymaRef: klog.ObjectRef{
			Name:      "5e32a9dd-4e68-47c7-aac7-64a4880a00d7",
			Namespace: infra.KCP().Namespace(),
		},
	}

	// Create DSL
	infra.InfraDSL = &infraDSL{i: infra}
	for _, c := range infra.clusters {
		c.ClusterDSL = &clusterDSL{
			ci:  c,
			ctx: infra.Ctx,
		}
	}

	_ = os.Setenv("GARDENER_NAMESPACE", infra.Garden().Namespace())

	return infra, nil
}

func startCluster(crdsDirs []string, projectRoot, envtestK8sVersion string) (*envtest.Environment, *rest.Config, error) {
	env := &envtest.Environment{
		CRDDirectoryPaths:     crdsDirs,
		ErrorIfCRDPathMissing: true,
		BinaryAssetsDirectory: filepath.Join(projectRoot, "bin", "k8s",
			fmt.Sprintf("%s-%s-%s", envtestK8sVersion, goruntime.GOOS, goruntime.GOARCH)),
	}

	cfg, err := env.Start()

	return env, cfg, err
}