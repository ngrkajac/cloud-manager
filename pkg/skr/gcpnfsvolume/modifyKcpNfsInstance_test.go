package gcpnfsvolume

import (
	"context"
	"github.com/go-logr/logr"
	cloudcontrolv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-control/v1beta1"
	cloudresourcesv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-resources/v1beta1"
	"github.com/kyma-project/cloud-manager/pkg/composed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
)

type modifyKcpNfsInstanceSuite struct {
	suite.Suite
	ctx context.Context
}

func (suite *modifyKcpNfsInstanceSuite) SetupTest() {
	suite.ctx = log.IntoContext(context.Background(), logr.Discard())
}

func (suite *modifyKcpNfsInstanceSuite) TestCreateNfsInstance() {
	factory, err := newTestStateFactory()
	assert.Nil(suite.T(), err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	state := factory.newState()
	state.KcpIpRange = &kcpIpRange

	//Invoke modifyKcpNfsInstance
	err, _ctx := modifyKcpNfsInstance(ctx, state)

	//validate expected return values
	assert.Equal(suite.T(), err, composed.StopWithRequeue)
	assert.Nil(suite.T(), _ctx)

	//Get the modified GcpNfsVolume object
	nfsVol := &cloudresourcesv1beta1.GcpNfsVolume{}
	err = factory.skrCluster.K8sClient().Get(ctx,
		types.NamespacedName{Name: gcpNfsVolume.Name, Namespace: gcpNfsVolume.Namespace}, nfsVol)

	//validate Status.ID of the GcpNfsVolume
	assert.Nil(suite.T(), err)
	assert.NotEqual(suite.T(), gcpNfsInstance.Name, gcpNfsVolume.Status.Id)

	//Get the KcpNfsInstance using theGcpNfsVolume.Status.Id
	nfsInstance := cloudcontrolv1beta1.NfsInstance{}
	err = factory.kcpCluster.K8sClient().Get(ctx,
		types.NamespacedName{Name: gcpNfsVolume.Status.Id, Namespace: kymaRef.Namespace}, &nfsInstance)
	assert.Nil(suite.T(), err)

	//Validate KCP NfsInstance labels.
	assert.Contains(suite.T(), nfsInstance.Labels, cloudcontrolv1beta1.LabelKymaName)
	assert.Equal(suite.T(), kymaRef.Name, nfsInstance.Labels[cloudcontrolv1beta1.LabelKymaName])
	assert.Contains(suite.T(), nfsInstance.Labels, cloudcontrolv1beta1.LabelRemoteName)
	assert.Equal(suite.T(), gcpNfsVolume.Name, nfsInstance.Labels[cloudcontrolv1beta1.LabelRemoteName])
	assert.Contains(suite.T(), nfsInstance.Labels, cloudcontrolv1beta1.LabelRemoteNamespace)
	assert.Equal(suite.T(), gcpNfsVolume.Namespace, nfsInstance.Labels[cloudcontrolv1beta1.LabelRemoteNamespace])

	//Validate KCPNfsInstance attributes.
	assert.Equal(suite.T(), kymaRef.Name, nfsInstance.Spec.Scope.Name)
	assert.Equal(suite.T(), gcpNfsVolume.Name, nfsInstance.Spec.RemoteRef.Name)
	assert.Equal(suite.T(), gcpNfsVolume.Namespace, nfsInstance.Spec.RemoteRef.Namespace)
	assert.Equal(suite.T(), kcpIpRange.Name, nfsInstance.Spec.IpRange.Name)
	assert.Equal(suite.T(), gcpNfsVolume.Spec.CapacityGb, nfsInstance.Spec.Instance.Gcp.CapacityGb)
	assert.Equal(suite.T(), string(gcpNfsVolume.Spec.Tier), string(nfsInstance.Spec.Instance.Gcp.Tier))
	assert.Equal(suite.T(), gcpNfsVolume.Spec.Location, nfsInstance.Spec.Instance.Gcp.Location)
	assert.Equal(suite.T(), gcpNfsVolume.Spec.FileShareName, nfsInstance.Spec.Instance.Gcp.FileShareName)
	assert.Equal(suite.T(), gcpNfsInstance.Spec.Instance.Gcp.ConnectMode, nfsInstance.Spec.Instance.Gcp.ConnectMode)
}

func (suite *modifyKcpNfsInstanceSuite) TestModifyNfsInstance() {
	factory, err := newTestStateFactory()
	assert.Nil(suite.T(), err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//Get state object with GcpNfsVolume
	nfsVol := gcpNfsVolume.DeepCopy()
	state := factory.newStateWith(nfsVol)
	state.KcpIpRange = &kcpIpRange
	state.KcpNfsInstance = &gcpNfsInstance

	//Update GcpNfsVolume with new CapacityGb
	nfsVol.Spec.CapacityGb = 2048
	err = factory.skrCluster.K8sClient().Update(ctx, nfsVol)
	assert.Nil(suite.T(), err)

	//Invoke modifyKcpNfsInstance
	err, _ctx := modifyKcpNfsInstance(ctx, state)

	//validate expected return values
	assert.Equal(suite.T(), err, composed.StopWithRequeue)
	assert.Nil(suite.T(), _ctx)

	//Get the KcpNfsInstance using theGcpNfsVolume.Status.Id
	nfsInstance := cloudcontrolv1beta1.NfsInstance{}
	err = factory.kcpCluster.K8sClient().Get(ctx,
		types.NamespacedName{Name: nfsVol.Status.Id, Namespace: kymaRef.Namespace}, &nfsInstance)
	assert.Nil(suite.T(), err)

	//Validate KCPNfsInstance attributes.
	assert.Equal(suite.T(), nfsVol.Spec.CapacityGb, nfsInstance.Spec.Instance.Gcp.CapacityGb)
}

func (suite *modifyKcpNfsInstanceSuite) TestWhenNfsVolumeDeleting() {
	factory, err := newTestStateFactory()
	assert.Nil(suite.T(), err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//Get state object with GcpNfsVolume
	state := factory.newStateWith(&deletedGcpNfsVolume)

	err, _ctx := modifyKcpNfsInstance(ctx, state)

	//validate expected return values
	assert.Nil(suite.T(), err)
	assert.Nil(suite.T(), _ctx)
}

func (suite *modifyKcpNfsInstanceSuite) TestWhenNfsVolumeNotChanged() {
	factory, err := newTestStateFactory()
	assert.Nil(suite.T(), err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//Get state object with GcpNfsVolume
	state := factory.newState()
	state.KcpNfsInstance = &gcpNfsInstance

	err, _ctx := modifyKcpNfsInstance(ctx, state)

	//validate expected return values
	assert.Nil(suite.T(), err)
	assert.Nil(suite.T(), _ctx)
}

func TestModifyKcpNfsInstance(t *testing.T) {
	suite.Run(t, new(modifyKcpNfsInstanceSuite))
}
