# Mutating admission controller webhook

Mutating admission controller automates injection of network resources into Kubernetes pods.

## Quickstart guide

Execute below steps to quickly build and deploy mutating webhook application:
```
make webhook
kubectl apply -f deployments/webhook/rbac.yaml
kubectl apply -f deployments/webhook/install.yaml
kubectl apply -f deployments/webhook/server.yaml
```

## Installation guide

### Building Docker image
Go to the root directory of sriov-network-device-plugin and execute:
```
cd $GOPATH/src/github.com/intel/sriov-network-device-plugin
make webhook
```

### Deploying webhook application
Create Service Account for mutating webhook and its installer and apply RBAC rules to created account:
```
kubectl apply -f deployments/webhook/rbac.yaml
```

Next step runs Kubernetes Job which creates all resources required to run webhook:
* mutating webhook configuration
* secret containing TLS key and certificate
* service to expose webhook deployment to the API server

Execute command:
```
kubectl apply -f deployments/webhook/install.yaml
```
*Note: Verify that Kubernetes controller manager has --cluster-signing-cert-file and --cluster-signing-key-file parameters set to paths to your CA keypair
to make sure that Certificates API is enabled in order to generate certificate signed by cluster CA.
More details about TLS certificates management in a cluster available [here](https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/).*

If Job has succesfully completed, you can run the actual webhook application.

Create webhook server Deployment:
```
kubectl apply -f deployments/webhook/server.yaml
```

