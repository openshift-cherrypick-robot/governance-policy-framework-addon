// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	"open-cluster-management.io/governance-policy-propagator/test/utils"
)

var _ = Describe("Test status sync", func() {
	const (
		case2PolicyName string = "case2-test-policy"
		case2PolicyYaml string = "../resources/case2_status_sync/case2-test-policy.yaml"
	)

	BeforeEach(func() {
		hubApplyPolicy(case2PolicyName, case2PolicyYaml)
	})
	AfterEach(func() {
		By("Deleting a policy on hub cluster in ns:" + clusterNamespaceOnHub)
		_, err := kubectlHub("delete", "-f", case2PolicyYaml, "-n", clusterNamespaceOnHub)
		Expect(err).ToNot(HaveOccurred())
		opt := metav1.ListOptions{}
		utils.ListWithTimeout(clientHubDynamic, gvrPolicy, opt, 0, true, defaultTimeoutSeconds)
		utils.ListWithTimeout(clientManagedDynamic, gvrPolicy, opt, 0, true, defaultTimeoutSeconds)
	})
	It("Should set status to compliant", func() {
		By("Generating an compliant event on the policy")
		managedPlc := utils.GetWithTimeout(
			clientManagedDynamic,
			gvrPolicy,
			case2PolicyName,
			clusterNamespace,
			true,
			defaultTimeoutSeconds)
		Expect(managedPlc).NotTo(BeNil())
		managedRecorder.Event(
			managedPlc,
			"Normal",
			"policy: managed/case2-test-policy-configurationpolicy",
			"Compliant; No violation detected")
		By("Checking if policy status is compliant")
		Eventually(func() interface{} {
			managedPlc = utils.GetWithTimeout(
				clientManagedDynamic,
				gvrPolicy,
				case2PolicyName,
				clusterNamespace,
				true,
				defaultTimeoutSeconds)

			return getCompliant(managedPlc)
		}, defaultTimeoutSeconds, 1).Should(Equal("Compliant"))
		By("Checking if hub policy status is in sync")
		Eventually(func() interface{} {
			hubPlc := utils.GetWithTimeout(
				clientHubDynamic,
				gvrPolicy,
				case2PolicyName,
				clusterNamespaceOnHub,
				true,
				defaultTimeoutSeconds)

			return hubPlc.Object["status"]
		}, defaultTimeoutSeconds, 1).Should(Equal(managedPlc.Object["status"]))
	})
	It("Should set status to NonCompliant", func() {
		By("Generating a noncompliant event on the policy")
		managedPlc := utils.GetWithTimeout(
			clientManagedDynamic,
			gvrPolicy,
			case2PolicyName,
			clusterNamespace,
			true,
			defaultTimeoutSeconds)
		Expect(managedPlc).NotTo(BeNil())
		managedRecorder.Event(
			managedPlc,
			"Warning",
			"policy: managed/case2-test-policy-configurationpolicy",
			"NonCompliant; there is violation")

		By("Checking if policy status is noncompliant")
		Eventually(checkCompliance(case2PolicyName), defaultTimeoutSeconds, 1).Should(Equal("NonCompliant"))

		By("Checking if policy history is correct")
		managedPlc = utils.GetWithTimeout(
			clientManagedDynamic,
			gvrPolicy,
			case2PolicyName,
			clusterNamespace,
			true,
			defaultTimeoutSeconds)
		var plc *policiesv1.Policy
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(managedPlc.Object, &plc)
		Expect(err).ToNot(HaveOccurred())
		Expect((plc.Status.Details)).To(HaveLen(1))
		Expect((plc.Status.Details[0].History)).To(HaveLen(1))
		Expect(plc.Status.Details[0].TemplateMeta.GetName()).To(Equal("case2-test-policy-configurationpolicy"))
		By("Checking if hub policy status is in sync")
		Eventually(func() interface{} {
			hubPlc := utils.GetWithTimeout(
				clientHubDynamic,
				gvrPolicy,
				case2PolicyName,
				clusterNamespaceOnHub,
				true,
				defaultTimeoutSeconds)

			return hubPlc.Object["status"]
		}, defaultTimeoutSeconds, 1).Should(Equal(managedPlc.Object["status"]))
	})
	It("Should set status to Compliant again", func(ctx SpecContext) {
		By("Generating an compliant event on the policy")
		managedPlc := utils.GetWithTimeout(
			clientManagedDynamic,
			gvrPolicy,
			case2PolicyName,
			clusterNamespace,
			true,
			defaultTimeoutSeconds)
		Expect(managedPlc).NotTo(BeNil())
		managedRecorder.Event(
			managedPlc,
			"Normal",
			"policy: managed/case2-test-policy-configurationpolicy",
			"Compliant; No violation detected")

		By("Checking if policy status is compliant")
		Eventually(checkCompliance(case2PolicyName), defaultTimeoutSeconds, 1).Should(Equal("Compliant"))

		By("Checking if policy history is correct")
		managedPlc = utils.GetWithTimeout(
			clientManagedDynamic,
			gvrPolicy,
			case2PolicyName,
			clusterNamespace,
			true,
			defaultTimeoutSeconds)
		var plc *policiesv1.Policy
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(managedPlc.Object, &plc)
		Expect(err).ToNot(HaveOccurred())
		Expect((plc.Status.Details)).To(HaveLen(1))
		Expect((plc.Status.Details[0].History)).To(HaveLen(1))
		Expect(plc.Status.Details[0].TemplateMeta.GetName()).To(Equal("case2-test-policy-configurationpolicy"))

		By("Checking if the hub policy status is in sync")
		var hubPlc *unstructured.Unstructured

		Eventually(func() interface{} {
			hubPlc = utils.GetWithTimeout(
				clientHubDynamic,
				gvrPolicy,
				case2PolicyName,
				clusterNamespaceOnHub,
				true,
				defaultTimeoutSeconds)

			return hubPlc.Object["status"]
		}, defaultTimeoutSeconds, 1).Should(Equal(managedPlc.Object["status"]))

		By("Clearing the hub status to verify it gets recovered")
		delete(hubPlc.Object, "status")
		_, err = clientHubDynamic.Resource(gvrPolicy).Namespace(clusterNamespaceOnHub).UpdateStatus(
			ctx, hubPlc, metav1.UpdateOptions{},
		)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() interface{} {
			hubPlc = utils.GetWithTimeout(
				clientHubDynamic,
				gvrPolicy,
				case2PolicyName,
				clusterNamespaceOnHub,
				true,
				defaultTimeoutSeconds)

			return hubPlc.Object["status"]
		}, defaultTimeoutSeconds, 1).Should(Equal(managedPlc.Object["status"]))

		By("clean up all events")
		_, err = kubectlManaged("delete", "events", "-n", clusterNamespace, "--all")
		Expect(err).ShouldNot(HaveOccurred())
	})
	It("Should hold up to last 10 history", func() {
		By("Generating an a lot of event on the policy")
		managedPlc := utils.GetWithTimeout(
			clientManagedDynamic,
			gvrPolicy,
			case2PolicyName,
			clusterNamespace,
			true,
			defaultTimeoutSeconds)
		Expect(managedPlc).NotTo(BeNil())
		historyIndex := 1
		for historyIndex < 12 {
			if historyIndex%2 == 0 {
				managedRecorder.Event(
					managedPlc,
					"Normal",
					"policy: managed/case2-test-policy-configurationpolicy",
					fmt.Sprintf("Compliant; No violation detected %d", historyIndex))
			} else {
				managedRecorder.Event(
					managedPlc,
					"Warning",
					"policy: managed/case2-test-policy-configurationpolicy",
					fmt.Sprintf("NonCompliant; there is violation %d", historyIndex))
			}
			historyIndex++
			utils.Pause(1)
		}
		var plc *policiesv1.Policy
		By("Generating a no violation event")
		managedRecorder.Event(
			managedPlc,
			"Normal",
			"policy: managed/case2-test-policy-configurationpolicy",
			"Compliant; No violation assert")
		Eventually(func(g Gomega) interface{} {
			managedPlc = utils.GetWithTimeout(
				clientManagedDynamic,
				gvrPolicy,
				case2PolicyName,
				clusterNamespace,
				true,
				defaultTimeoutSeconds)
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(managedPlc.Object, &plc)
			g.Expect(err).ToNot(HaveOccurred())

			if len(plc.Status.Details) == 0 {
				return "status.details[] is empty"
			}
			if len(plc.Status.Details[0].History) == 0 {
				return "status.details[0].history[] is empty"
			}

			return plc.Status.Details[0].History[0].Message
		}, defaultTimeoutSeconds, 1).Should(Equal("Compliant; No violation assert"))
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(managedPlc.Object, &plc)
		Expect(err).ToNot(HaveOccurred())
		By("Checking no violation event is the first one in history")
		Expect(plc.Status.ComplianceState).To(Equal(policiesv1.Compliant))
		Expect((plc.Status.Details)).To(HaveLen(1))
		By("Checking if size of the history is 10")
		Expect(plc.Status.Details[0].History).To(HaveLen(10))
		Expect(plc.Status.Details[0].TemplateMeta.GetName()).To(Equal("case2-test-policy-configurationpolicy"))
		By("Generating a violation event")
		managedRecorder.Event(
			managedPlc,
			"Warning",
			"policy: managed/case2-test-policy-configurationpolicy",
			"NonCompliant; Violation assert")
		Eventually(func(g Gomega) interface{} {
			managedPlc = utils.GetWithTimeout(
				clientManagedDynamic,
				gvrPolicy,
				case2PolicyName,
				clusterNamespace,
				true,
				defaultTimeoutSeconds)
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(managedPlc.Object, &plc)
			g.Expect(err).ToNot(HaveOccurred())

			return plc.Status.Details[0].History[0].Message
		}, defaultTimeoutSeconds, 1).Should(Equal("NonCompliant; Violation assert"))
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(managedPlc.Object, &plc)
		Expect(err).ToNot(HaveOccurred())
		By("Checking violation event is the first one in history")
		Expect(plc.Status.ComplianceState).To(Equal(policiesv1.NonCompliant))
		Expect((plc.Status.Details)).To(HaveLen(1))
		By("Checking if size of the history is 10")
		Expect(plc.Status.Details[0].History).To(HaveLen(10))
		Expect(plc.Status.Details[0].TemplateMeta.GetName()).To(Equal("case2-test-policy-configurationpolicy"))
		By("Checking if hub policy status is in sync")
		Eventually(func() interface{} {
			hubPlc := utils.GetWithTimeout(
				clientHubDynamic,
				gvrPolicy,
				case2PolicyName,
				clusterNamespaceOnHub,
				true,
				defaultTimeoutSeconds)

			return hubPlc.Object["status"]
		}, defaultTimeoutSeconds, 1).Should(Equal(managedPlc.Object["status"]))
		By("clean up all events")
		_, err = kubectlManaged(
			"delete",
			"events",
			"-n",
			clusterNamespace,
			"--all",
		)
		Expect(err).ShouldNot(HaveOccurred())
	})
})
