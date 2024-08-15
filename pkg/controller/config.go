package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *LoadBalancerReconciler) ensureConfigMap(ctx context.Context, serviceDN, name, namespace string, ports []corev1.ServicePort) error {
	yaml := fmt.Sprintf(`
serverAddr: %s
serverPort: %d
auth:
  method: token
  token: %s
proxies:
`, RemoteServerHostName, RemoteServerPort, RemoteServerAuthKey)

	for i, servicePort := range ports {
		if servicePort.Protocol != corev1.ProtocolTCP {
			continue
		}

		port := servicePort.Port

		yaml += fmt.Sprintf(`- name: "port-%d"
  type: "tcp"
  localIP: "%s"
  localPort: %d
  remotePort: %d
`,
			i, serviceDN, port, port)
	}

	data := map[string]string{
		"config.yaml": yaml,
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Data = data
		return nil
	})
	return err
}
