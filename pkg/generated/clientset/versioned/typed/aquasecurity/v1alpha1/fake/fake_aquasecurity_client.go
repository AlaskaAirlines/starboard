// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/aquasecurity/starboard/pkg/generated/clientset/versioned/typed/aquasecurity/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeAquasecurityV1alpha1 struct {
	*testing.Fake
}

func (c *FakeAquasecurityV1alpha1) CISKubeBenchReports() v1alpha1.CISKubeBenchReportInterface {
	return &FakeCISKubeBenchReports{c}
}

func (c *FakeAquasecurityV1alpha1) ConfigAuditReports(namespace string) v1alpha1.ConfigAuditReportInterface {
	return &FakeConfigAuditReports{c, namespace}
}

func (c *FakeAquasecurityV1alpha1) KubeHunterReports() v1alpha1.KubeHunterReportInterface {
	return &FakeKubeHunterReports{c}
}

func (c *FakeAquasecurityV1alpha1) Vulnerabilities(namespace string) v1alpha1.VulnerabilityInterface {
	return &FakeVulnerabilities{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeAquasecurityV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}