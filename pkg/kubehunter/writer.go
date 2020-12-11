package kubehunter

import (
	"context"

	starboard "github.com/AlaskaAirlines/starboard/pkg/apis/aquasecurity/v1alpha1"
)

type Writer interface {
	Write(ctx context.Context, report starboard.KubeHunterOutput, cluster string) error
}
