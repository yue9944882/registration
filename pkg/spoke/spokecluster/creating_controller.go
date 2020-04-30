package spokecluster

import (
	"context"
	"fmt"
	"time"

	clientset "github.com/open-cluster-management/api/client/cluster/clientset/versioned"
	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type setHasSpokeClusterFunc func(bool)

// spokeClusterCreatingController creates a spoke cluster on hub cluster during the spoke agent bootstrap phase
type spokeClusterCreatingController struct {
	clusterName            string
	spokeExternalServerUrl string
	spokeCABundle          []byte
	setHasSpokeCluster     setHasSpokeClusterFunc
	hubClusterClient       clientset.Interface
}

// NewSpokeClusterCreatingController creates a new spoke cluster creating controller on the spoke cluster.
func NewSpokeClusterCreatingController(
	clusterName, spokeExternalServerUrl string,
	spokeCABundle []byte,
	hubClusterClient clientset.Interface,
	setHasSpokeClusterFn setHasSpokeClusterFunc,
	recorder events.Recorder) factory.Controller {
	c := &spokeClusterCreatingController{
		clusterName:            clusterName,
		spokeExternalServerUrl: spokeExternalServerUrl,
		spokeCABundle:          spokeCABundle,
		setHasSpokeCluster:     setHasSpokeClusterFn,
		hubClusterClient:       hubClusterClient,
	}
	return factory.New().
		WithSync(c.sync).
		ResyncEvery(5*time.Minute).
		ToController("SpokeClusterCreatingController", recorder)
}

func (c *spokeClusterCreatingController) sync(ctx context.Context, syncCtx factory.SyncContext) error {
	_, err := c.hubClusterClient.ClusterV1().SpokeClusters().Get(ctx, c.clusterName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		spokeCluster := &clusterv1.SpokeCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: c.clusterName,
			},
			Spec: clusterv1.SpokeClusterSpec{
				SpokeClientConfig: clusterv1.ClientConfig{
					URL:      c.spokeExternalServerUrl,
					CABundle: c.spokeCABundle,
				},
			},
		}
		_, err := c.hubClusterClient.ClusterV1().SpokeClusters().Create(ctx, spokeCluster, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("unable to create spoke cluster with name %q on hub: %w", c.clusterName, err)
		}
		c.setHasSpokeCluster(true)
		syncCtx.Recorder().Eventf("SpokeClusterCreated", "Spoke cluster %q created on hub", c.clusterName)
		return nil
	}

	if err != nil {
		return err
	}

	c.setHasSpokeCluster(true)
	return nil
}
