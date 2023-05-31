# AlloyDB Auth Proxy Sidecar

The instructions for deploying an auth proxy sidecar for AlloyDB are very similar
to those for deploying a Cloud SQL Auth Proxy Sidecar. Before starting, make sure
you have a working AlloyDB instance.

## Before you begin

1. Enable access to AlloyDB in your project by following these [instructions](https://cloud.google.com/alloydb/docs/project-enable-access)

1. Create a VPC network and [configure Private Services Access for AlloyDB](https://cloud.google.com/alloydb/docs/configure-connectivity).
Make note of the VPC name.

1. Create a [Serverless VPC Connector](https://cloud.google.com/run/docs/configuring/connecting-vpc#yaml)
for Cloud Run. Make note of the connector name.

1. Create an AlloyDB cluster and its primary instance by following these [instructions](https://cloud.google.com/alloydb/docs/cluster-create).
Make note of the Cluster ID, Instance ID, IP Address and Password

1. Create a database for your application by following these 
[instructions](https://cloud.google.com/alloydb/docs/database-create).
Note the database name.

1. Create a user in your database by following these
[instructions](https://cloud.google.com/alloydb/docs/database-users/about).
Note the username.

## Deploying the application

The excerpt below demonstrates how to create a connection pool which connects to
AlloyDB:

```ruby
require 'sinatra'
require 'sequel'

set :bind, '0.0.0.0'
set :port, 8080

# Configure a connection pool that connects to the proxy via TCP
def connect_tcp
    Sequel.connect(
        adapter: 'postgres',
        host: ENV["INSTANCE_HOST"],
        database: ENV["DB_NAME"],
        user: ENV["DB_USER"],
        password: ENV["DB_PASS"],
        pool_timeout: 5,
        max_connections: 5,
    )
end

DB = connect_tcp()
```

 Next, build the container image for the main application and deploy it:

```bash
gcloud builds submit --tag gcr.io/<YOUR_PROJECT_ID>/run-alloydb
```

Finally, update the `multicontainer.yaml` file with the correct values for your
deployment for `VPC_CONNECTOR_NAME` `YOUR_PROJECT_ID`, `DB_USER`, `DB_PASS`, `DB_NAME`, and `INSTANCE_URI`, listing the AlloyDB container image as a sidecar:

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  annotations: 
     run.googleapis.com/launch-stage: ALPHA
  name: multicontainer-service
spec:
  template:
    metadata:
      annotations:
        run.googleapis.com/execution-environment: gen1 #or gen2
        run.googleapis.com/vpc-access-connector: <VPC_CONNECTOR_NAME>
    spec:
      containers:
      - name: my-app
        image: gcr.io/<YOUR_PROJECT_ID>/run-alloydb
        ports:
          - containerPort: 8080
       env:
          - name: DB_USER
            value: <DB_USER>
          - name: DB_PASS
            value: <DB_PASS>
          - name: DB_NAME
            value: <DB_NAME>
          - name: INSTANCE_HOST
            value: "127.0.0.1"
          - name: DB_PORT
            value: "5432"
      - name: alloydb-auth-proxy
        image: gcr.io/alloydb-connectors/alloydb-auth-proxy:latest
        args:

             # Ensure the port number on the --port argument matches the value of the DB_PORT env var on the my-app container.
             - "--port=5432"
            # Instance URIs follow the format 
            # projects/PROJECT_ID/locations/REGION_ID/clusters/CLUSTER_ID/instances/INSTANCE_ID
             - "<INSTANCE_URI>"
```

You can optionally use Secret Manager to store the database password. See
[this documentation](https://cloud.google.com/run/docs/deploying#yaml) for more details.

Before deploying, you will need to make sure that the service account associated
with the Cloud Run Deployment has the AlloyDB Client role. See [this documentation](https://cloud.google.com/alloydb/docs/reference/iam-roles-permissions)
for more details. The default service account will already have these permissions.

Finally, you can deploy the service using:

```bash
gcloud run services replace multicontainer.yaml
```

Once the service is deployed, the console should print out a URL. You can test
the service by sending a curl request with your gcloud identity token in the headers:

```bash
curl -H \
"Authorization: Bearer $(gcloud auth print-identity-token)" \
<SERVICE_URL>
```
