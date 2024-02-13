package cloudcontrol

import (
	cloudcontrolv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-control/v1beta1"
	iprangePkg "github.com/kyma-project/cloud-manager/pkg/kcp/iprange"
	scopePkg "github.com/kyma-project/cloud-manager/pkg/kcp/scope"
	. "github.com/kyma-project/cloud-manager/pkg/testinfra/dsl"
	"github.com/kyma-project/cloud-manager/pkg/util/debugged"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var _ = Describe("Created KCP NfsVolume for GCP is projected into FileStore and it gets Ready condition", func() {

	Context("Given KCP Cluster", Ordered, func() {

		const (
			kymaName = "5b30a61a-c4ae-49da-a8ad-903a71696d8b"

			interval = time.Millisecond * 250
		)
		var (
			timeout = time.Second * 10
		)

		if debugged.Debugged {
			timeout = time.Minute * 20
		}

		// Tell Scope reconciler to ignore this kymaName
		scopePkg.Ignore.AddName(kymaName)
		scope := &cloudcontrolv1beta1.Scope{}
		It("And Given KCP Scope exists", func() {

			// Given Scope exists
			Expect(
				infra.GivenScopeGcpExists(kymaName),
			).NotTo(HaveOccurred())

			// Load created scope
			Eventually(func() (exists bool, err error) {
				err = infra.KCP().Client().Get(infra.Ctx(), infra.KCP().ObjKey(kymaName), scope)
				exists = err == nil
				return exists, client.IgnoreNotFound(err)
			}, timeout, interval).
				Should(BeTrue(), "expected Scope to get created")
		})
		kcpIpRangeName := "gcp-nfs-iprange-1"
		kcpIpRange := &cloudcontrolv1beta1.IpRange{}

		iprangePkg.Ignore.AddName(kcpIpRangeName)
		It("And Given KCP IPRange exists", func() {
			// Tell IpRange reconciler to ignore this kymaName
			Eventually(CreateKcpIpRange).
				WithArguments(
					infra.Ctx(), infra.KCP().Client(), kcpIpRange,
					WithName(kcpIpRangeName),
					WithKcpIpRangeSpecScope(scope.Name),
				).
				Should(Succeed())
		})

		It("And Given KCP IpRange has Ready condition", func() {
			Eventually(UpdateStatus).
				WithArguments(
					infra.Ctx(), infra.KCP().Client(), kcpIpRange,
					WithKcpIpRangeStatusCidr(kcpIpRange.Spec.Cidr),
					WithConditions(KcpReadyCondition()),
				).WithTimeout(timeout).WithPolling(interval).
				Should(Succeed())
		})

		gcpNfsInstance := &cloudcontrolv1beta1.NfsInstance{}

		It("When KCP NfsVolume is created", func() {
			Eventually(CreateNfsInstance).
				WithArguments(
					infra.Ctx(), infra.KCP().Client(), gcpNfsInstance,
					WithName("gcp-nfs-instance-1"),
					WithNfsInstanceScope(scope.Name),
					WithNfsInstanceIpRange(kcpIpRange.Name),
					WithNfsInstanceGcp(scope.Spec.Region),
				).
				Should(Succeed())
		})
		It("Then KCP NfsVolume will get Ready condition", func() {

			// Then IpRange will get Ready condition
			Eventually(func() (exists bool, err error) {
				err = infra.KCP().Client().Get(infra.Ctx(), client.ObjectKeyFromObject(gcpNfsInstance), gcpNfsInstance)
				if err != nil {
					return false, err
				}
				exists = meta.IsStatusConditionTrue(gcpNfsInstance.Status.Conditions, cloudcontrolv1beta1.ConditionTypeReady)
				return exists, nil
			}, timeout, interval).
				Should(BeTrue(), "expected NfsInstance for GCP with Ready condition")
		})
		It("And Then KCP NfsVolume has Ready state", func() {
			Expect(gcpNfsInstance.Status.State).To(Equal(cloudcontrolv1beta1.ReadyState))
		})
	})
})
