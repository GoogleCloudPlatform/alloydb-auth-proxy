# Connecting to AlloyDB

## Before you begin

1. Enable access to AlloyDB in your project by following these [instructions](https://cloud.google.com/alloydb/docs/project-enable-access)

1. Create a VPC network and [configure Private Services Access for AlloyDB](https://cloud.google.com/alloydb/docs/configure-connectivity)

1. Create an AlloyDB cluster and its primary instance by following these [instructions](https://cloud.google.com/alloydb/docs/cluster-create). Make note of the Cluster ID, Instance ID, IP Address and Password

1. Create a database for your application by following these 
[instructions](https://cloud.google.com/alloydb/docs/database-create). Note the database
name. 

1. Create a user in your database by following these 
[instructions](https://cloud.google.com/alloydb/docs/database-users/about). Note the username. 

1. Create a [service account](https://cloud.google.com/iam/docs/understanding-service-accounts) with the 'AlloyDB Client' permissions.


1. Use the information noted in the previous steps:
```bash
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service/account/key.json
export DB_USER='<YOUR_DB_USER_NAME>'
export DB_PASS='<YOUR_DB_PASSWORD>'
export DB_NAME='<YOUR_DB_NAME>'
export INSTANCE_HOST='<IP Address of Cluster or 127.0.0.1 if using auth proxy>'
export DB_POST=5432
```
Note: Saving credentials in environment variables is convenient, but not secure - consider a more
secure solution such as [Secret Manager](https://cloud.google.com/secret-manager/) to help keep secrets safe.


## Deploy to App Engine Standard

To run on GAE-Standard, create an App Engine project by following the setup for these 
[instructions](https://cloud.google.com/appengine/docs/standard/python3/quickstart#before-you-begin).

First, update `app.standard.yaml` with the correct values to pass the environment 
variables into the runtime. Your `app.standard.yaml` file should look like this:

```yaml
runtime: python37
entrypoint: gunicorn -b :$PORT app:app

env_variables:
  INSTANCE_HOST: '<IP Address of Cluster>'
  DB_PORT: 5432
  DB_USER: <YOUR_DB_USER_NAME>
  DB_PASS: <YOUR_DB_PASSWORD>
  DB_NAME: <YOUR_DB_NAME>
```

Note: Saving credentials in environment variables is convenient, but not secure - consider a more
secure solution such as [Secret Manager](https://cloud.google.com/secret-manager/docs/overview) to
help keep secrets safe.

Next, the following command will deploy the application to your Google Cloud project:

```bash
gcloud app deploy app.standard.yaml
```

## Deploy to App Engine Flexible

To run on GAE-Flexible, create an App Engine project by following the setup for these 
[instructions](https://cloud.google.com/appengine/docs/flexible/python/quickstart#before-you-begin).

First, update `app.flexible.yaml` with the correct values to pass the environment 
variables into the runtime. Your `app.flexible.yaml` file should look like this:

```yaml
runtime: custom
env: flex
entrypoint: gunicorn -b :$PORT app:app

env_variables:
  INSTANCE_HOST: '<IP Address of Cluster>'
  DB_PORT: 5432
  DB_USER: <YOUR_DB_USER_NAME>
  DB_PASS: <YOUR_DB_PASSWORD>
  DB_NAME: <YOUR_DB_NAME>

```

Note: Saving credentials in environment variables is convenient, but not secure - consider a more
secure solution such as [Secret Manager](https://cloud.google.com/secret-manager/docs/overview) to
help keep secrets safe.

Next, the following command will deploy the application to your Google Cloud project:

```bash
gcloud app deploy app.flexible.yaml
```

## Deploy to Cloud Run

Before deploying the application, you will need to [configure a Serverless VPC Connector](https://cloud.google.com/vpc/docs/configure-serverless-vpc-access) to be able to connect to the VPC in which your AlloyDB cluster is running.
1. Build the container image:

```sh
gcloud builds submit --tag gcr.io/[YOUR_PROJECT_ID]/run-alloydb
```

2. Deploy the service to Cloud Run:

  ```sh
  gcloud run deploy run-alloydb \
    --image gcr.io/[YOUR_PROJECT_ID]/run-alloydb \
    --platform managed \
    --vpc-connector=[YOUR_VPC_CONNECTOR] \
    --allow-unauthenticated \
    --region [REGION] \
    --update-env-vars INSTANCE_HOST=[INSTANCE_HOST] \
    --update-env-vars DB_PORT=[DB_PORT] \
    --update-env-vars DB_USER=[MY_DB_USER] \
    --update-env-vars DB_PASS=[MY_DB_PASS] \
    --update-env-vars DB_NAME=[MY_DB]
  ```

Take note of the URL output at the end of the deployment process.

Replace environment variables with the correct values for your AlloyDB
instance configuration.

It is recommended to use the [Secret Manager integration](https://cloud.google.com/run/docs/configuring/secrets) for Cloud Run instead
of using environment variables for the AlloyDB configuration. The service injects the AlloyDB credentials from
Secret Manager at runtime via an environment variable.

Create secrets via the command line:
```sh
echo -n $DB_USER | \
    gcloud secrets versions add DB_USER_SECRET --data-file=-
```

Deploy the service to Cloud Run specifying the env var name and secret name:
```sh
gcloud beta run deploy SERVICE --image gcr.io/[YOUR_PROJECT_ID]/run-alloydb \
    --update-secrets INSTANCE_HOST=[INSTANCE_HOST_SECRET]:latest,\
      DB_PORT=[DB_PORT_SECRET]:latest, \
      DB_USER=[DB_USER_SECRET]:latest, \
      DB_PASS=[DB_PASS_SECRET]:latest, \
      DB_NAME=[DB_NAME_SECRET]:latest
```

3. Navigate your browser to the URL noted in step 2.

For more details about using Cloud Run see http://cloud.run.
Review other [Python on Cloud Run samples](../../../run/).

## Deploy to Cloud Functions

To deploy the service to [Cloud Functions](https://cloud.google.com/functions/docs) run the following command:

```sh
gcloud functions deploy votes --runtime python39 --trigger-http --allow-unauthenticated \
--set-env-vars INSTANCE_HOST=$INSTANCE_HOST \
--set-env-vars DB_PORT=$DB_PORT \
--set-env-vars DB_USER=$DB_USER \
--set-env-vars DB_PASS=$DB_PASS \
 --set-env-vars DB_NAME=$DB_NAME
```

Take note of the URL output at the end of the deployment process or run the following to view your function:

```sh
gcloud app browse
```
