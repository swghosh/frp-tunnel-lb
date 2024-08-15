package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	frpv1client "github.com/fatedier/frp/pkg/config/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ProxyConfigExt struct {
	// eg.
	// name = "wireguard"
	// type = "udp"
	// localIP = "127.0.0.1"
	// localPort = 51820
	// remotePort = 51820

	Name       string `json:"name"`
	Type       string `json:"type"`
	LocalIP    string `json:"localIP"`
	LocalPort  int    `json:"localPort"`
	RemotePort int    `json:"remotePort"`
}

// clientConfigExt extends the `frp.ClientConfig` to support the correct marshal function.
type clientConfigExt struct {
	frpv1client.ClientCommonConfig `json:",inline"`
	Proxies                        []ProxyConfigExt `json:"proxies"`
}

func generateFRPCJsonConfig(serviceDN string, ports []corev1.ServicePort) (string, error) {
	serverPort, err := strconv.Atoi(RemoteServerPort)
	if err != nil {
		return "", err
	}

	config := &clientConfigExt{
		ClientCommonConfig: frpv1client.ClientCommonConfig{
			Auth: frpv1client.AuthClientConfig{
				Method: frpv1client.AuthMethodToken,
				Token:  RemoteServerAuthKey,
			},
			ServerAddr: RemoteServerHostName,
			ServerPort: serverPort,
		},
		Proxies: make([]ProxyConfigExt, 0),
	}

	for i, servicePort := range ports {
		proxyConfig := ProxyConfigExt{}
		if servicePort.Protocol == corev1.ProtocolTCP {
			proxyConfig.Type = "tcp"
		} else if servicePort.Protocol == corev1.ProtocolUDP {
			proxyConfig.Type = "udp"
		} else {
			// today we can only support TCP and UDP yet
			continue
		}

		proxyConfig.Name = fmt.Sprintf("port-%d", i)
		proxyConfig.LocalIP = serviceDN
		proxyConfig.LocalPort = int(servicePort.Port)
		proxyConfig.RemotePort = int(servicePort.Port)
		config.Proxies = append(config.Proxies, proxyConfig)
	}

	proxyConfigAsJson, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(proxyConfigAsJson), nil
}

func (r *LoadBalancerReconciler) ensureConfigMap(ctx context.Context, serviceDN, name, namespace string, ports []corev1.ServicePort) error {
	frpcConfig, err := generateFRPCJsonConfig(serviceDN, ports)
	if err != nil {
		return err
	}

	data := map[string]string{
		"config.json": frpcConfig,
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Data = data
		return nil
	})
	return err
}
