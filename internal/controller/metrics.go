package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	metrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var findingsGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "healthpolicy_findings",
		Help: "Current number of findings per policy and rule type",
	},
	[]string{"policy", "namespace", "rule_type"},
)

func init() {
	metrics.Registry.MustRegister(findingsGauge)
}
