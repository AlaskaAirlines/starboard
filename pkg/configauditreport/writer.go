package configauditreport

import (
	"context"

	clientset "github.com/AlaskaAirlines/starboard/pkg/generated/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"

	starboard "github.com/AlaskaAirlines/starboard/pkg/apis/aquasecurity/v1alpha1"
	"github.com/AlaskaAirlines/starboard/pkg/kube"
)

// Write is the interface that wraps basic methods for persisting ConfigAudit reports.
//
// Write persists the given ConfigAuditReport report.
type Writer interface {
	Write(ctx context.Context, report starboard.ConfigAuditReport) error
}

// Reader is the interface that wraps basic methods for reading ConfigAudit reports.
//
// Read will return a single ConfigAuditReport that match a specific workload
type Reader interface {
	FindByOwner(ctx context.Context, owner kube.Object) (*starboard.ConfigAuditReport, error)
}

type ReadWriter interface {
	Writer
	Reader
}

type readWriter struct {
	client clientset.Interface
}

func NewReadWriter(client clientset.Interface) ReadWriter {
	return &readWriter{
		client: client,
	}
}

func (w *readWriter) Write(ctx context.Context, report starboard.ConfigAuditReport) error {
	existing, err := w.client.AquasecurityV1alpha1().ConfigAuditReports(report.Namespace).
		Get(ctx, report.Name, metav1.GetOptions{})

	if err == nil {
		klog.V(3).Infof("Updating ConfigAuditReport %q", report.Namespace+"/"+report.Name)
		deepCopy := existing.DeepCopy()
		deepCopy.Labels = report.Labels
		deepCopy.Report = report.Report

		_, err = w.client.AquasecurityV1alpha1().ConfigAuditReports(report.Namespace).
			Update(ctx, deepCopy, metav1.UpdateOptions{})
		return err
	}

	if errors.IsNotFound(err) {
		klog.V(3).Infof("Creating ConfigAuditReport %q", report.Namespace+"/"+report.Name)
		_, err = w.client.AquasecurityV1alpha1().ConfigAuditReports(report.Namespace).
			Create(ctx, &report, metav1.CreateOptions{})
		return err
	}

	return err
}

func (w *readWriter) FindByOwner(ctx context.Context, workload kube.Object) (*starboard.ConfigAuditReport, error) {
	list, err := w.client.AquasecurityV1alpha1().ConfigAuditReports(workload.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.Set{
			kube.LabelResourceKind:      string(workload.Kind),
			kube.LabelResourceName:      workload.Name,
			kube.LabelResourceNamespace: workload.Namespace,
		}.String(),
	})
	if err != nil {
		return nil, err
	}
	// Only one config audit per specific workload exists on the cluster
	if len(list.Items) > 0 {
		return &list.DeepCopy().Items[0], nil
	}
	return nil, nil
}
