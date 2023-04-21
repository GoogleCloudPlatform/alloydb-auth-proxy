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
export DB_USER='my-db-user'
export DB_PASS='my-db-pass'
export DB_NAME='my_db'
export DB_HOST='<IP Address of Cluster or 127.0.0.1 if using auth proxy>'
export DB_PORT=5432
export ALLOYDB_CONNECTION_NAME='projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>'
```
Note: Saving credentials in environment variables is convenient, but not secure - consider a more
secure solution such as [Secret Manager](https://cloud.google.com/secret-manager/) to help keep secrets safe.


## Google App Engine Standard

To run on GAE-Standard, create an AppEngine project by following the setup for these 
[instructions](https://cloud.google.com/appengine/docs/standard/java/quickstart#before-you-begin) 
and verify that 
[appengine-maven-plugin](https://cloud.google.com/java/docs/setup#optional_install_maven_or_gradle_plugin_for_app_engine)
 has been added in your build section as a plugin.


### Deploy to Google App Engine
Before deploying the application, you will need to [configure a Serverless VPC Connector](https://cloud.google.com/vpc/docs/configure-serverless-vpc-access) to be able to connect to the VPC in which your AlloyDB cluster is running.

First, update `src/main/webapp/WEB-INF/appengine-web.xml` with the correct values to pass the 
environment variables into the runtime.

Next, the following command will deploy the application to your Google Cloud project:
```bash
mvn clean package appengine:deploy
```

### Deploy to Cloud Run
Before deploying the application, you will need to [configure a Serverless VPC Connector](https://cloud.google.com/vpc/docs/configure-serverless-vpc-access) to be able to connect to the VPC in which your AlloyDB cluster is running.

1. Build the container image using [Jib](https://cloud.google.com/java/getting-started/jib):

  ```sh
mvn clean package com.google.cloud.tools:jib-maven-plugin:2.8.0:build \
 -Dimage=gcr.io/[YOUR_PROJECT_ID]/run-postgres -DskipTests
  ```

2. Deploy the service to Cloud Run:

  ```sh
  gcloud run deploy run-postgres \
    --image gcr.io/[YOUR_PROJECT_ID]/run-alloydb \
    --vpc-connector=[YOUR_VPC_CONNECTOR] \
    --platform managed \
    --allow-unauthenticated \
    --region [REGION] \
    --update-env-vars ALLOYDB_CONNECTION_NAME=[ALLOYDB_CONNECTION_NAME] \
    --update-env-vars DB_USER=[MY_DB_USER] \
    --update-env-vars DB_PASS=[MY_DB_PASS] \
    --update-env-vars DB_NAME=[MY_DB]
  ```

  Replace environment variables with the correct values for your AlloyDB instance configuration.

  Take note of the URL output at the end of the deployment process.

  It is recommended to use the [Secret Manager integration](https://cloud.google.com/run/docs/configuring/secrets) for Cloud Run instead
  of using environment variables for the AlloyDB configuration. The service injects the Alloy credentials from
  Secret Manager at runtime via an environment variable.

  Create secrets via the command line:
  ```sh
  echo -n "projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>" | \
      gcloud secrets versions add ALLOYDB_CONNECTION_NAME_SECRET --data-file=-
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
  Review other [Java on Cloud Run samples](../../../run/).

### Deploy to Google Cloud Functions
Before deploying the application, you will need to [configure a Serverless VPC Connector](https://cloud.google.com/vpc/docs/configure-serverless-vpc-access) to be able to connect to the VPC in which your AlloyDB cluster is running.

To deploy the application to Cloud Functions, first fill in the values for required environment variables in `.env.yaml`. Then run the following command
```
gcloud functions deploy alloydb-sample \
  --trigger-http \
  --entry-point com.example.alloydb.functions.Main \
  --runtime java11 \
  --env-vars-file .env.yaml
```
