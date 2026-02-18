# Appliance VM

This VM is built on flatcar linux and k3s.

## Quickstart

The following commands are for local development using Vagrant.

Note: The VM is provisioned with a 2nd NIC (eth1) with an IP of 192.168.56.10. If you need a different
IP provisioned, you can set it with `export K8S_IP=x.x.x.x` before running vagrant up.

```bash
vagrant up
vagrant ssh control1 -- -t 'sudo cat /etc/rancher/k3s/k3s.yaml' > k3s.yaml
export KUBECONFIG=$PWD/k3s.yaml
k get nodes
NAME       STATUS   ROLES                       AGE     VERSION
control1   Ready    control-plane,etcd,master   8m19s   v1.30.0+k3s1
```
