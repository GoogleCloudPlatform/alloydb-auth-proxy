# Connecting to AlloyDB from a Go web app

This repo contains the Go source code for a simple web app that can be deployed to App Engine Standard. It is a demonstration of how to connect to AlloyDB cluster.

## Before you begin

1. Enable access to AlloyDB in your project by following these [instructions](https://cloud.google.com/alloydb/docs/project-enable-access)

1. Create a VPC network and [configure Private Services Access for AlloyDB](https://cloud.google.com/alloydb/docs/configure-connectivity)

1. Create an AlloyDB cluster and its primary instance by following these [instructions](https://cloud.google.com/alloydb/docs/cluster-create). Make note of the Cluster ID, Instance ID, IP Address and Password

1. Create a database for your application by following these 
[instructions](https://cloud.google.com/alloydb/docs/database-create). Note the database
name. 

1. Create a user in your database by following these 
[instructions](https://cloud.google.com/alloydb/docs/database-users/about). Note the username. 

1. [Create a service account](https://cloud.google.com/iam/docs/creating-managing-service-accounts#creating)
and then grant that service acount the 'AlloyDB Client' permissions by following these 
[instructions](https://cloud.google.com/alloydb/docs/user-grant-access#procedure).
Download the service account's JSON key to use to authenticate your connection. 

1. Use the information noted in the previous steps:
```bash
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service/account/key.json
export DB_USER='<YOUR_DB_USER_NAME>'
export DB_PASS='<YOUR_DB_PASSWORD>'
export DB_NAME='<YOUR_DB_NAME>'
export DB_HOST='<IP Address of Cluster or 127.0.0.1 if using auth proxy>'
export DB_POST=5432
export ALLOYDB_CONNECTION_NAME='projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>'
```
Note: Saving credentials in environment variables is convenient, but not secure - consider a more
secure solution such as [Secret Manager](https://cloud.google.com/secret-manager/) to help keep secrets safe.

## Deploying to App Engine Standard

To run the sample on GAE-Standard, create an App Engine project by following the setup for these
[instructions](https://cloud.google.com/appengine/docs/standard/go/quickstart#before-you-begin).

First, update [`app.standard.yaml`](cmd/app/app.standard.yaml) with the correct values to pass the environment
variables into the runtime. Your `app.standard.yaml` file should look like this:

```yaml
runtime: go116
env_variables:
  DB_HOST: 127.0.0.1
  DB_PORT: 5432
  DB_USER: <YOUR_DB_USER_NAME>
  DB_PASS: <YOUR_DB_PASSWORD>
  DB_NAME: <YOUR_DB_NAME>
```

Note: Saving credentials in environment variables is convenient, but not secure - consider a more
secure solution such as [Cloud Secret Manager](https://cloud.google.com/secret-manager) to help keep secrets safe.

Next, the following command will deploy the application to your Google Cloud project:

```bash
gcloud app deploy cmd/app/app.standard.yaml
```


## Deploy to Cloud Run

Before deploying the application, you will need to [configure a Serverless VPC Connector](https://cloud.google.com/vpc/docs/configure-serverless-vpc-access) to be able to connect to the VPC in which your AlloyDB cluster is running.

1. Build the container image:

```sh
gcloud builds submit --tag gcr.io/[YOUR_PROJECT_ID]/run-alloydb
```

2. Deploy the service to Cloud Run:

```sh
gcloud run deploy run-alloydb --image gcr.io/[YOUR_PROJECT_ID]/run-alloydb \
  --platform managed \
  --vpc-connector=[YOUR_VPC_CONNECTOR] \
  --vpc-egress=all-traffic \
  --allow-unauthenticated \
  --region <REGION> \
  --update-env-vars ALLOYDB_CONNECTION_NAME=<ALLOYDB_CONNECTION_NAME> \
  --update-env-vars DB_USER='<YOUR_DB_USER_NAME>' \
  --update-env-vars DB_PASS='<YOUR_DB_PASSWORD>' \
  --update-env-vars DB_NAME='<YOUR_DB_NAME>'
```

Take note of the URL output at the end of the deployment process.

Replace environment variables with the correct values for your AlloyDB configuration.

It is recommended to use the [Secret Manager integration](https://cloud.google.com/run/docs/configuring/secrets) for Cloud Run instead
of using environment variables for the SQL configuration. The service injects the AlloyDB credentials from
Secret Manager at runtime via an environment variable.

Create secrets via the command line:
```sh
echo -n $ALLOYDB_CONNECTION_NAME | \
    gcloud secrets create [ALLOYDB_CONNECTION_NAME_SECRET] --data-file=-
```

Deploy the service to Cloud Run specifying the env var name and secret name:
```sh
gcloud beta run deploy SERVICE --image gcr.io/[YOUR_PROJECT_ID]/run-alloydb \
  --update-secrets ALLOYDB_CONNECTION_NAME=[ALLOYDB_CONNECTION_NAME_SECRET]:latest,\
    DB_USER=[DB_USER_SECRET]:latest, \
    DB_PASS=[DB_PASS_SECRET]:latest, \
    DB_NAME=[DB_NAME_SECRET]:latest
```

3. Navigate your browser to the URL noted in step 2.

For more details about using Cloud Run see http://cloud.run.
