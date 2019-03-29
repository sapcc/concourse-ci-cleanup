package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/concourse/concourse/go-concourse/concourse"
	"github.com/golang/glog"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	ONLY_DELETE_VOLUMES_IN_STATE = "available"
	ONLY_DELETE_VOLUMES_OLDER    = time.Hour * 1
)

var (
	kubeconfigFile    string
	kubeContext       string
	nodes             *v1.NodeList
	concourseURL      string
	concourseUser     string
	concoursePassword string
	concourseClient   concourse.Client
	workerPrefix      string
	volumePrefix      string
	runVolumeCleanup  bool
	o                 *OpenstackClient
)

func init() {
	var err error

	concourseURL = os.Getenv("CONCOURSE_URL")
	concourseUser = os.Getenv("CONCOURSE_USER")
	workerPrefix = os.Getenv("WORKER_PREFIX")

	runVolumeCleanup, err = strconv.ParseBool(os.Getenv("VOLUME_CLEANUP"))
	if err != nil {
		runVolumeCleanup = false
	}
	volumePrefix = os.Getenv("VOLUME_PREFIX")

	o = NewOpenstackClient()
	o.IdentityEndpoint = os.Getenv("OS_AUTH_URL")
	o.ApplicationCredentialID = os.Getenv("OS_APPLICATION_CREDENTIAL_ID")

	flag.StringVar(&kubeconfigFile, "kubeconfig", "", "Use explicit kubeconfig file")
	flag.StringVar(&kubeContext, "context", "", "Use context")
	flag.StringVar(&concourseURL, "concourse-url", concourseURL, "Use concourse URL [CONCOURSE_URL]")
	flag.StringVar(&concourseUser, "concourse-user", concourseUser, "Use concourse URL [CONCOURSE_USER]")
	flag.StringVar(&concoursePassword, "concourse-password", "", "Use concourse URL [CONCOURSE_PASSWORD]")
	flag.StringVar(&workerPrefix, "worker-prefix", workerPrefix, "Prefix to identify stale workers [WORKER_PREFIX]")
	flag.BoolVar(&runVolumeCleanup, "volume-cleanup", runVolumeCleanup, "Cleanup volumes in Openstack [VOLUME_CLEANUP]")
	flag.StringVar(&volumePrefix, "volume-prefix", volumePrefix, "Prefix to identify unused volumes [VOLUME_PREFIX]")
	flag.StringVar(&o.IdentityEndpoint, "os-auth-url", o.IdentityEndpoint, "Openstack auth url [OS_AUTH_URL]")
	flag.StringVar(&o.ApplicationCredentialID, "os-application-credential-id", o.ApplicationCredentialID, "Openstack application credential id [OS_APPLICATION_CREDENTIAL_ID]")
	flag.StringVar(&o.ApplicationCredentialSecret, "os-application-credential-secret", "", "Openstack application credential secret [OS_APPLICATION_CREDENTIAL_SECRET]")

	if concoursePassword == "" {
		concoursePassword = os.Getenv("CONCOURSE_PASSWORD")
	}

	if o.ApplicationCredentialSecret == "" {
		o.ApplicationCredentialSecret = os.Getenv("OS_APPLICATION_CREDENTIAL_SECRET")
	}
}

func sigHandler() <-chan struct{} {
	stop := make(chan struct{})
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c,
			syscall.SIGINT,  // Ctrl+C
			syscall.SIGTERM) // Termination Request
		sig := <-c
		glog.Warningf("Signal (%v) Detected, Shutting Down", sig)
		close(stop)
	}()
	return stop
}

func main() {
	flag.Parse()

	if concourseURL == "" || concourseUser == "" || concoursePassword == "" {
		glog.Fatal("Needed flag is missing: -concourse-url, -concourse-user or -concourse-password")
	}

	if runVolumeCleanup && (o.IdentityEndpoint == "" || o.ApplicationCredentialID == "" || o.ApplicationCredentialSecret == "") {
		glog.Fatal("Needed flag is missing: -os-auth-url, -os-application-credential-id or -os-application-credential-secret")
	}

	kubeconfig, err := kubeConfig(kubeconfigFile, kubeContext)
	if err != nil {
		glog.Fatalf("Failed to create kubeconfig: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		glog.Fatalf("Could not create client set: %v", err)
	}

	nodes, err = clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		glog.Fatalf("Could not get node list: %v", err)
	}

	concourseClient, err = NewConcourseClient(concourseURL, concourseUser, concoursePassword)
	if err != nil {
		glog.Errorf("Could not create concourse client: %v", err)
	}

	glog.Info("Going to cleanup stale workers ...")
	err = cleanupStaleWorkers()
	if err != nil {
		glog.Error(err)
	}

	if runVolumeCleanup {
		err := o.Setup()
		if err != nil {
			glog.Fatalf("Failed to setup Openstack client: %v", err)
		}

		glog.Info("Going to cleanup volumes ...")
		err = cleanupVolumes()
		if err != nil {
			glog.Error(err)
		}
	}
}

func cleanupVolumes() error {
	storageClient, err := openstack.NewBlockStorageV3(o.Provider, gophercloud.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("Could not create block storage client: %v", err)
	}

	project, err := tokens.Get(o.Identity, o.Provider.Token()).ExtractProject()
	if err != nil {
		return fmt.Errorf("There should be no error while extracting the project: %v", err)
	}

	volumeListOpts := volumes.ListOpts{
		TenantID: project.ID,
	}

	allPages, err := volumes.List(storageClient, volumeListOpts).AllPages()
	if err != nil {
		return fmt.Errorf("There should be no error while retrieving volume pages: %v", err)
	}

	allVolumes, err := volumes.ExtractVolumes(allPages)
	if err != nil {
		return fmt.Errorf("There should be no error while extracting volumes: %v", err)
	}

	for _, vol := range allVolumes {
		node := vol.Metadata["concourse-worker"]
		if strings.HasPrefix(vol.Name, volumePrefix) &&
			vol.Status == ONLY_DELETE_VOLUMES_IN_STATE &&
			vol.Metadata["concourse-team"] != "" &&
			time.Since(vol.CreatedAt) > ONLY_DELETE_VOLUMES_OLDER &&
			!inNodeList(node) {
			err := volumes.Delete(storageClient, vol.ID, volumes.DeleteOpts{}).ExtractErr()
			if err != nil {
				glog.Errorf("There should be no error while deleting volume %s (%s)", vol.Name, vol.ID)
			} else {
				glog.Infof("Volume %s (%s) has been deleted.", vol.Name, vol.ID)
			}
		}
	}

	return nil
}

func cleanupStaleWorkers() error {
	workers, err := concourseClient.ListWorkers()
	if err != nil {
		return fmt.Errorf("Could not get list of workers: %v", err)
	}

	for _, worker := range workers {
		if (worker.State == "stalled" || worker.State == "landed") && strings.HasPrefix(worker.Name, workerPrefix) && !inNodeList(worker.Name) {
			err := concourseClient.PruneWorker(worker.Name)
			if err != nil {
				glog.Errorf("Could not prune stale worker %s: %v", worker.Name, err)
			} else {
				glog.Infof("Stale worker %s has been pruned.", worker.Name)
			}
		}
	}

	return nil
}

func inNodeList(nodeName string) bool {
	for _, node := range nodes.Items {
		if nodeName == node.Name {
			return true
		}
	}

	return false
}

func kubeConfig(kubeconfig, context string) (*rest.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}

	if len(context) > 0 {
		overrides.CurrentContext = context
	}

	if len(kubeconfig) > 0 {
		rules.ExplicitPath = kubeconfig
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
}
