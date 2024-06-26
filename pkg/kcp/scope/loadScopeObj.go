package scope

import (
	"context"
	cloudcontrolv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-control/v1beta1"
	"github.com/kyma-project/cloud-manager/pkg/composed"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func loadScopeObj(ctx context.Context, st composed.State) (error, context.Context) {
	state := st.(*State)

	err := state.LoadObj(ctx)
	if apierrors.IsNotFound(err) {
		list := &cloudcontrolv1beta1.ScopeList{}
		err = state.Cluster().K8sClient().List(ctx, list)

		// continue to create one
		return nil, nil
	}
	if err != nil {
		return composed.LogErrorAndReturn(err, "Error loading Scope object", composed.StopWithRequeue, ctx)
	}

	return nil, nil
}
