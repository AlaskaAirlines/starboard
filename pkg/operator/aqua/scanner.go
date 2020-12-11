package aqua

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/AlaskaAirlines/s/starboard/pkg/docker"
	"github.com/AlaskaAirlines/s/starboard/pkg/ext"
	"github.com/AlaskaAirlines/s/starboard/pkg/operator/etc"
	"github.com/AlaskaAirlines/s/starboard/pkg/starboard"
	"github.com/AlaskaAirlines/s/starboard/pkg/vulnerabilityreport"
	"github.com/AlaskaAirlines/starboard/pkg/apis/aquasecurity/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"
)

const (
	secretName = "starboard-operator"
)

type scanner struct {
	idGenerator ext.IDGenerator
	buildInfo   starboard.BuildInfo
	config      etc.ScannerAquaCSP
}

// NewScanner constructs a new vulnerability scanner Plugin, which is using
// the Aqua CSP to scan container images of Kubernetes workloads.
func NewScanner(
	idGenerator ext.IDGenerator,
	buildInfo starboard.BuildInfo,
	config etc.ScannerAquaCSP,
) vulnerabilityreport.Plugin {
	return &scanner{
		idGenerator: idGenerator,
		buildInfo:   buildInfo,
		config:      config,
	}
}

func (s *scanner) GetScanJobSpec(spec corev1.PodSpec, _ map[string]docker.Auth) (corev1.PodSpec, []*corev1.Secret, error) {
	initContainerName := s.idGenerator.GenerateID()

	scanJobContainers := make([]corev1.Container, len(spec.Containers))
	for i, container := range spec.Containers {
		var err error
		scanJobContainers[i], err = s.newScanJobContainer(container)
		if err != nil {
			return corev1.PodSpec{}, nil, err
		}
	}

	return corev1.PodSpec{
		RestartPolicy:                corev1.RestartPolicyNever,
		AutomountServiceAccountToken: pointer.BoolPtr(false),
		NodeName:                     spec.NodeName,
		Volumes: []corev1.Volume{
			{
				Name: "scannercli",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "dockersock",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/run/docker.sock",
					},
				},
			},
		},
		InitContainers: []corev1.Container{
			{
				Name:  initContainerName,
				Image: s.config.ImageRef,
				Command: []string{
					"cp",
					"/opt/aquasec/scannercli",
					"/downloads/scannercli",
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "scannercli",
						MountPath: "/downloads",
					},
				},
			},
		},
		Containers: scanJobContainers,
	}, nil, nil
}

func (s *scanner) newScanJobContainer(podContainer corev1.Container) (corev1.Container, error) {
	version, err := starboard.GetVersionFromImageRef(s.config.ImageRef)
	if err != nil {
		return corev1.Container{}, err
	}

	return corev1.Container{
		Name:            podContainer.Name,
		Image:           fmt.Sprintf("aquasec/starboard-scanner-aqua:%s", s.buildInfo.Version),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{
			"/bin/sh",
			"-c",
			fmt.Sprintf("/usr/local/bin/starboard-scanner-aqua --version $(AQUA_VERSION) --host $(AQUA_CSP_HOST) --user $(AQUA_CSP_USERNAME) --password $(AQUA_CSP_PASSWORD) %s 2> %s",
				podContainer.Image,
				corev1.TerminationMessagePathDefault),
		},
		Env: []corev1.EnvVar{
			{
				Name:  "AQUA_VERSION",
				Value: version,
			},
			{
				Name: "AQUA_CSP_HOST",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secretName,
						},
						Key: "OPERATOR_SCANNER_AQUA_CSP_HOST",
					},
				},
			},
			{
				Name: "AQUA_CSP_USERNAME",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secretName,
						},
						Key: "OPERATOR_SCANNER_AQUA_CSP_USERNAME",
					},
				},
			},
			{
				Name: "AQUA_CSP_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secretName,
						},
						Key: "OPERATOR_SCANNER_AQUA_CSP_PASSWORD",
					},
				},
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("100M"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("500M"),
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "scannercli",
				MountPath: "/usr/local/bin/scannercli",
				SubPath:   "scannercli",
			},
			{
				Name:      "dockersock",
				MountPath: "/var/run/docker.sock",
			},
		},
	}, nil
}

func (s *scanner) ParseVulnerabilityScanResult(_ string, logsReader io.ReadCloser) (v1alpha1.VulnerabilityScanResult, error) {
	var report v1alpha1.VulnerabilityScanResult
	err := json.NewDecoder(logsReader).Decode(&report)
	return report, err
}
