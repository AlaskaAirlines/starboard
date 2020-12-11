// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	versioned "github.com/AlaskaAirlines/s/starboard/pkg/generated/clientset/versioned"
	internalinterfaces "github.com/AlaskaAirlines/s/starboard/pkg/generated/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/AlaskaAirlines/s/starboard/pkg/generated/listers/aquasecurity/v1alpha1"
	aquasecurityv1alpha1 "github.com/AlaskaAirlines/starboard/pkg/apis/aquasecurity/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// CISKubeBenchReportInformer provides access to a shared informer and lister for
// CISKubeBenchReports.
type CISKubeBenchReportInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.CISKubeBenchReportLister
}

type cISKubeBenchReportInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewCISKubeBenchReportInformer constructs a new informer for CISKubeBenchReport type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewCISKubeBenchReportInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredCISKubeBenchReportInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredCISKubeBenchReportInformer constructs a new informer for CISKubeBenchReport type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredCISKubeBenchReportInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AquasecurityV1alpha1().CISKubeBenchReports().List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AquasecurityV1alpha1().CISKubeBenchReports().Watch(context.TODO(), options)
			},
		},
		&aquasecurityv1alpha1.CISKubeBenchReport{},
		resyncPeriod,
		indexers,
	)
}

func (f *cISKubeBenchReportInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredCISKubeBenchReportInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *cISKubeBenchReportInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&aquasecurityv1alpha1.CISKubeBenchReport{}, f.defaultInformer)
}

func (f *cISKubeBenchReportInformer) Lister() v1alpha1.CISKubeBenchReportLister {
	return v1alpha1.NewCISKubeBenchReportLister(f.Informer().GetIndexer())
}
