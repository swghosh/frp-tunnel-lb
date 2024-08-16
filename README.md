# TunnelLB: Kubernetes LoadBalancer controller using reverse proxy tunnels
Expose any k8s Service with zero networking config to the public internet or any network outside the k8s cluster easily. Minimal Kubernetes Load Balancer Controller that works using [fast-reverse-proxy](https://github.com/fatedier/frp) tunnels.

Reverse proxy forwards public traffic from outside the cluster to service inside the cluster with no network configuration at all, just Kubernetes pods. Suitable for edge devices running k8s where cloud load balancers cannot be configured and or for networks where service provider's inbound configuration falls short.

More details, coming soon! **FYI: implementation is buggy, do not use in production yet.**

## How to quickly use this?

1. **Tunnel Infra**: Setup a VM (with public external IP, firewall allow traffic on all ports):

eg.
```bash
zone=asia-south1-a
vm_type=e2-medium
vm_name="frps-vm"

frps_port="10240"
frps_token="<put-your-fav-password>"

gcloud compute instances create-with-container "${vm_name}" \
    --project=openshift-gce-devel \
    --zone=${zone} \
    --machine-type=${vm_type} \
    --image="projects/cos-cloud/global/images/cos-stable-109-17800-147-22" \
    --boot-disk-size=10GB \
    --boot-disk-type=pd-balanced \
    --boot-disk-device-name=vm-disk \
    --container-image="quay.io/swghosh/fast-reverse-proxy:latest" \
    --container-restart-policy=always \
    --container-arg=--bind-port=${frps_port} \
    --container-arg=--kcp-bind-port=${frps_port} \
    --container-arg=--token="${frps_token}"
```

2. **Tunnel Routing**: Run this inside a pod in the target Kubernetes cluster (it needs access to r/w Service, r/w Deployment, r/w ConfigMaps only):

```bash
go install github.com/swghosh/frp-tunnel-lb

FRPS_SERVER_HOST="<vm-ip-or-host>" \
 FRPS_SERVER_AUTH_KEY="<put-your-fav-password>" \
 FRPS_SERVER_PORT="10240" \
 FRP_IMAGE="quay.io/swghosh/fast-reverse-proxy:latest" \
 FRP_EXPOSED_HOST="<vm-ip-or-host>" \
 frp-tunnel-lb
```

3. **Ingress/Gateway/...**: Use an ingress-controller of your choice (for traffic management inside the cluster), or even Gateway API to distribute traffic through HTTP/TCP Routes. All such implementations are dependent on Kubernetes Service of type LoadBalancer and our controller seamlessly adds the ExternalIP traffic routing to the load balancer service pods.
