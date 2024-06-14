# Connecting to AlloyDB from Google Kubernetes Engine

This example shows you how to configure a deployment to connect to AlloyDB from
Google Kubernetes Engine (GKE).

Depending on your connection path, you will configure a deployment file either
using [direct_connection_deployment.yaml][] or
[proxy_sidecar_deployment.yaml][]. Once that deployment file is filled out, you
will be ready to deploy a workload to GKE.

[direct_connection_deployment.yaml]: direct_connection_deployment.yaml
[proxy_sidecar_deployment.yaml]: proxy_sidecar_deployment.yaml

## Choosing between Auth Proxy Connections and Direct Connections

If you are using public IP, we recommend using the AlloyDB Auth Proxy.

If you are using private IP (either PSA or PSC), you may either connect
directly or use the Auth Proxy.

The Auth Proxy will provide a connection encrypted with mTLS 1.3, but requires
running a sidecar in you deployment.

The direct connection will provide a connection encrypted with TLS 1.3, but
does not support client certificates.

If you prefer the highest level of security with some inconvenience, use the
Auth Proxy. If you do not need the extra security provided by client
certificates, use a direct connection.

NOTE: for connecting over private IP, you will need to have a [VPC-native
cluster][vpc-native].

[vpc-native]: https://cloud.google.com/kubernetes-engine/docs/concepts/alias-ips

## Prerequisites

This guide assumes you have:

- basic working knowledge of Kubernetes, kubectl, deployments, etc.
- decided between using private and public IP for your AlloyDB instance
- enabled the AlloyDB API and created an AlloyDB instance
- created a GKE cluster (VPC Native or not) with Workload Identity enabled
- containerized an application that will connect to AlloyDB and that configures
  the Postgres DSN using environment variables for username, password, dbname,
  etc.

## Connecting Directly to AlloyDB

Let's configure a deployment to connect directly to AlloyDB.

> [!IMPORTANT]
> This section assumes you're using private IP or PSC from a VPC Native GKE
> Cluster. If you're either not using private IP or PSC with a VPC Native
> Cluster, you will not have network connectivity for this section to work.
> Skip to the section below on using the Auth Proxy.

We will use a [Kubernetes Secrets][ksa-secret] to store connection information.

[ksa-secret]: https://kubernetes.io/docs/concepts/configuration/secret/

```shell
kubectl create secret generic <YOUR-DB-SECRET> \
    --from-literal=username=<YOUR-DATABASE-USER> \
    --from-literal=password=<YOUR-DATABASE-PASSWORD> \
    --from-literal=database=<YOUR-DATABASE-NAME> \
    --from-literal=hostname=<YOUR-DATABASE-HOST>
```

The value for `hostname` may be the PSA IP address (e.g., 10.0.0.1) or the
PSC DNS record (e.g., INSTANCE_UID.PROJECT_UID.REGION_NAME.alloydb-psc.goog).

For example:

```
kubectl create secret generic mycoolsecret \
    --from-literal=username=myuser \
    --from-literal=password=mypassword \
    --from-literal=database=mydatabase \
    --from-literal=hostname=10.0.0.2
```

Next, we will update [direct_connection_deployment.yaml][] to source all the
connection-related environment variables from the Secret.

For example, if you created a Secret named "mycoolsecret," you would update the
`env` section like so:

```yaml
env:
- name: DB_USER
  valueFrom:
    secretKeyRef:
      name: mycoolsecret
      key: username
- name: DB_PASS
  valueFrom:
    secretKeyRef:
      name: mycoolsecret
      key: password
- name: DB_NAME
  valueFrom:
    secretKeyRef:
      name: mycoolsecret
      key: database
- name: DB_HOST
  valueFrom:
   secretKeyRef:
     name: mycoolsecret
     key: hostname
```

Lastly, update [direct_connection_deployment.yaml][] to reference your
containerized application by changing the placeholder
`<YOUR-APPLICATION-IMAGE-URL>` to the URL of your image. Additionally, change
`<YOUR-DEPLOYMENT-NAME>` to whatever you want to call the deployment (e.g.,
"mydeployment") and `<YOUR-APPLICATION-NAME>` (e.g., "mycoolapp").

You now have a fully-configured and secure deployment that connects to
AlloyDB with connection configuration stored in a Secret.

## Connecting Using the AlloyDB Auth Proxy

Let's configure a deployment to connect using the Auth Proxy as a sidecar.

We recommend using the Auth Proxy as a sidecar because, it:

- Ensures all unencrypted traffic between the application and the Proxy never
  leaves the VM.
- Prevents a single point of failure. Each application's access to your
  database is independent from the others, making it more resilient.
- Limits access to the proxy, allowing you to use IAM permissions per
  application.
- Allows you to scope resource requests more accurately. Using the Proxy as a
  sidecar allows you to more accurately scope and request resources to match
  your applications as it scales.

To make the Auth Proxy work in GKE, we will need to take a few additional
steps:

- Set up an IAM Service Account (SA) with the proper IAM permissions
- Create a Kubernetes Service Account (KSA)
- Set up Workload Identity so workloads run transparently as the Service
  Account
- Configure a deployment with the Auth Proxy as a sidecar.

### Setting up a Service Account

First, [create a Service Account in Cloud IAM][create-sa].

[create-sa]: https://cloud.google.com/iam/docs/service-accounts-create

> [!NOTE]
> It's a best practice to create a service account for each application. This
> follows the principle of least privilege.

The Service Account should be granted the [IAM permissions][iam] necessary for
the Proxy to run:

- roles/alloydb.client
- roles/serviceusage.serviceUsageConsumer

[iam]: https://cloud.google.com/alloydb/docs/auth-proxy/overview#how-authorized

### Workload Identity

For workloads in GKE to run as a Service Account, we need to enable Workload
Identity. If you created a GKE cluster, it's possible you already enabled
Workload Identity.

If you don't have Workload Identity enabled, or aren't sure, see [Authenticate
to Google Cloud APIs from GKE workloads][wi], which includes setup information
and steps to verify the setup.

[wi]: https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity

Next, follow the [instructions to create and configure a Kubernetes Service
Account (KSA)][ksa-setup], linking your KSA with an IAM service account that
you created above.

> [!NOTE]
> To use IAM Authentication in the future, you'll want to link a Kubernetes
> Service Account to an IAM Service Account. The default of using IAM principal
> identifiers to configure Workload Identity Federation for GKE is incompatible
> with AlloyDB's IAM Authentication.

[ksa-setup]: https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity#kubernetes-sa-to-iam

Next, make sure to specify the service account in
[proxy_sidecar_deployment.yaml][].

For example, if you created a KSA called `mycoolksa`, you would update the
deployment like so:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: <YOUR-DEPLOYMENT-NAME>
spec:
  selector:
    matchLabels:
      app: <YOUR-APPLICATION-NAME>
  template:
    metadata:
      labels:
        app: <YOUR-APPLICATION-NAME>
    spec:
      serviceAccountName: mycoolksa # <- KSA name goes here
      # ...
```

Next, update [proxy_sidecar_deployment.yaml][] to reference the secret.

For example, if you created a Secret named "mysecret," you would update the
`env` section like so:

```yaml
env:
- name: DB_USER
  valueFrom:
    secretKeyRef:
      name: mysecret
      key: username
- name: DB_PASS
  valueFrom:
    secretKeyRef:
      name: mysecret
      key: password
- name: DB_NAME
  valueFrom:
    secretKeyRef:
      name: mysecret
      key: database
```

Lastly, update [direct_connection_deployment.yaml][] to reference your
containerized application by changing the placeholder
`<YOUR-APPLICATION-IMAGE-URL>` to the URL of your image. Additionally, change
`<YOUR-DEPLOYMENT-NAME>` to whatever you want to call the deployment (e.g.,
"mydeployment") and `<YOUR-APPLICATION-NAME>` (e.g., "mycoolapp").

You now have a fully-configured and secure deployment that connects to
AlloyDB with connection configuration stored in a Secret.
