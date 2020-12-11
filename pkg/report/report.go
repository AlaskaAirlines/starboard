package report

import (
	"io"

	"github.com/AlaskaAirlines/starboard/pkg/kube"
)

type Reporter interface {
	GenerateReport(workload kube.Object, writer io.Writer) error
}
