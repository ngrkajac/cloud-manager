package gcpnfsvolume

import (
	"context"
	"fmt"
	"github.com/kyma-project/cloud-manager/api/cloud-resources/v1beta1"
	"github.com/kyma-project/cloud-manager/pkg/composed"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func createPersistenceVolume(ctx context.Context, st composed.State) (error, context.Context) {
	state := st.(*State)
	logger := composed.LoggerFromCtx(ctx)

	//If NfsVolume is marked for deletion, continue
	if composed.MarkedForDeletionPredicate(ctx, st) {
		// SKR GcpNfsVolume is NOT marked for deletion, do not delete mirror in KCP
		return nil, nil
	}

	//Get GcpNfsVolume object
	nfsVolume := state.ObjAsGcpNfsVolume()
	capacity := resource.NewQuantity(int64(nfsVolume.Spec.CapacityGb)*1024*1024*1024, resource.BinarySI)

	//If GcpNfsVolume is not Ready state, continue.
	if !meta.IsStatusConditionTrue(nfsVolume.Status.Conditions, v1beta1.ConditionTypeReady) {
		return nil, nil
	}

	//PV already exists, continue.
	if state.PV != nil {
		return nil, nil
	}

	//If the NFS Host list is empty, create error response.
	if len(nfsVolume.Status.Hosts) == 0 {
		logger.WithValues("kyma-name", state.KymaRef).
			WithValues("NfsVolume", state.ObjAsGcpNfsVolume().Name).
			Info("Error creating PV: Not able to get Host(s).")
		return nil, nil
	}

	//Construct a PV Object
	state.PV = &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:        getVolumeName(nfsVolume),
			Labels:      getVolumeLabels(nfsVolume),
			Annotations: getVolumeAnnotations(nfsVolume),
			Finalizers: []string{
				v1beta1.Finalizer,
			},
		},
		Spec: v1.PersistentVolumeSpec{
			Capacity: v1.ResourceList{
				"storage": *capacity,
			},
			AccessModes:      []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			StorageClassName: "",
			PersistentVolumeSource: v1.PersistentVolumeSource{
				NFS: &v1.NFSVolumeSource{
					Server: nfsVolume.Status.Hosts[0],
					Path:   fmt.Sprintf("/%s", nfsVolume.Spec.FileShareName),
				},
			},
		},
	}

	//Create PV
	err := state.SkrCluster.K8sClient().Create(ctx, state.PV)
	if err != nil {
		return composed.LogErrorAndReturn(err, "Error creating PersistentVolume", composed.StopWithRequeue, ctx)
	}

	//continue
	return composed.StopWithRequeueDelay(3 * time.Second), nil
}
