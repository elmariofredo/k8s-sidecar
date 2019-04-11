package main

import "github.com/prometheus/client_golang/prometheus"

var (
	sidecarSyntaxOk = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sidecar_syntax_ok",
			Help: "Sidecar Syntax OK.",
		},
		[]string{"namespace", "config"},
	)
)

func init() {
	prometheus.MustRegister(sidecarSyntaxOk)
	//sidecarSyntaxOk.WithLabelValues("namespace","config").Set(1)
}
