# AlloyDB Auth Proxy Sidecar

The instructions for deploying an auth proxy sidecar for AlloyDB are very similar
to those for deploying a Cloud SQL Auth Proxy Sidecar. Before starting, make sure
you have a working AlloyDB instance. Make note of the connection URI, and the
database name, username, and password needed for authentication.

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

Finally, create a revision YAML file (multicontainers.yaml), using the `example.yaml`
file as a referece for the deployment, listing the AlloyDB container image as a sidecar:

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
        image: gcr.io/cloud-sql-connectors/alloydb-auth-proxy:latest
        args:

             # Replace DB_PORT with the port the proxy should listen on
             - "--port=5432"
             - "<INSTANCE_URI>"
```

Before deploying, you will need to make sure that the service account associated
with the Cloud Run Deployment has the AlloyDB Client role. See [this documentation](https://cloud.google.com/alloydb/docs/reference/iam-roles-permissions)
for more details.

Finally, you can deploy the service using:

```bash
gcloud run services replace multicontainers.yaml
```
