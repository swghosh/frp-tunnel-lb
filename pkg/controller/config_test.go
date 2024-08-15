package controller

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestGenerateConfig(t *testing.T) {
	config, err := generateFRPCJsonConfig("service-a.namespace-b.svc", []corev1.ServicePort{
		{
			Protocol:   corev1.ProtocolTCP,
			Port:       80,
			TargetPort: intstr.FromInt32(8080),
			Name:       "eighty-eighty-tcp",
		},
		{
			Protocol:   corev1.ProtocolUDP,
			Port:       51820,
			TargetPort: intstr.FromInt32(51820),
			Name:       "wg-maybe",
		},
		{
			Protocol:   corev1.ProtocolSCTP,
			Port:       218,
			TargetPort: intstr.FromInt32(218),
			Name:       "notorius-sctp",
		},
	})
	t.Logf("%q", config)

	if err != nil {
		t.Fatal(err)
	}
}
