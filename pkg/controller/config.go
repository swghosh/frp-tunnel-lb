package controller

import (
	"context"
	"encoding/json"
	"strconv"

	frpv1client "github.com/fatedier/frp/pkg/config/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func generateFRPCJsonConfig(serviceDN string, ports []corev1.ServicePort) (string, error) {
	serverPort, err := strconv.Atoi(RemoteServerPort)
	if err != nil {
		return "", err
	}

	config := frpv1client.ClientConfig{
		ClientCommonConfig: frpv1client.ClientCommonConfig{
			Auth: frpv1client.AuthClientConfig{
				Method: frpv1client.AuthMethodToken,
				Token:  RemoteServerAuthKey,
			},
			ServerAddr: RemoteServerHostName,
			ServerPort: serverPort,
		},
	}

	proxies := []frpv1client.TypedProxyConfig{}
	for _, servicePort := range ports {
		var proxy frpv1client.TypedProxyConfig
		if servicePort.Protocol == corev1.ProtocolTCP {
			proxy = frpv1client.TypedProxyConfig{
				Type: "tcp",
				ProxyConfigurer: &frpv1client.TCPProxyConfig{
					ProxyBaseConfig: frpv1client.ProxyBaseConfig{
						ProxyBackend: frpv1client.ProxyBackend{
							LocalIP:   serviceDN,
							LocalPort: int(servicePort.Port),
						},
					},
					RemotePort: int(servicePort.Port),
				},
			}
		} else if servicePort.Protocol == corev1.ProtocolUDP {
			proxy = frpv1client.TypedProxyConfig{
				Type: "udp",
				ProxyConfigurer: &frpv1client.UDPProxyConfig{
					ProxyBaseConfig: frpv1client.ProxyBaseConfig{
						ProxyBackend: frpv1client.ProxyBackend{
							LocalIP:   serviceDN,
							LocalPort: int(servicePort.Port),
						},
					},
					RemotePort: int(servicePort.Port),
				},
			}
		} else {
			// today we only support TCP & UDP!
			continue
		}

		proxies = append(proxies, proxy)
	}

	config.Proxies = proxies

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
