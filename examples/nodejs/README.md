# Connecting to AlloyDB from a Node.js web app

## Before you begin

1. If you haven't already, set up a Node.js Development Environment by following the [Node.js setup guide](https://cloud.google.com/nodejs/docs/setup)  and
[create a project](https://cloud.google.com/resource-manager/docs/creating-managing-projects#creating_a_project).

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
export DB_USER='my-db-user'
export DB_PASS='my-db-pass'
export DB_NAME='my_db'
export DB_HOST='<IP Address of Cluster or 127.0.0.1 if using auth proxy>'
export DB_PORT=5432
```

Note: Saving credentials in environment variables is convenient, but not secure - consider a more
secure solution such as [Secret Manager](https://cloud.google.com/secret-manager/) to help keep secrets safe.

## Deploy to Google App Engine Standard

1. To allow your app to connect to your AlloyDB cluster when the app is deployed, add the user, password, and database name from AlloyDB to the related environment variables in the [`app.standard.yaml`](app.standard.yaml) file. The deployed application will connect via TCP sockets.

    ```yaml
    env_variables:
      DB_HOST: 127.0.0.1
      DB_PORT: 5432
      DB_USER: MY_DB_USER
      DB_PASS: MY_DB_PASSWORD
      DB_NAME: MY_DATABASE
    ```

2. To deploy to App Engine Standard, run the following command:

    ```
    gcloud app deploy app.standard.yaml
    ```

3. To launch your browser and view the app at https://[YOUR_PROJECT_ID].appspot.com, run the following command:

    ```
    gcloud app browse
    ```

## Deploy to Cloud Run

Before deploying the application, you will need to [configure a Serverless VPC Connector](https://cloud.google.com/vpc/docs/configure-serverless-vpc-access) to be able to connect to the VPC in which your AlloyDB cluster is running.

1. Build the container image:

```sh
gcloud builds submit --tag gcr.io/[YOUR_PROJECT_ID]/run-alloydb
```

2. Deploy the service to Cloud Run:

```sh
gcloud run deploy run-sql --image gcr.io/[YOUR_PROJECT_ID]/run-alloydb \
  --platform managed \
  --vpc-connector=[YOUR_VPC_CONNECTOR] \
  --allow-unauthenticated \
  --region [REGION] \
  --update-env-vars DB_HOST=[MY_DB_HOST] \
  --update-env-vars DB_PORT=[DB_PORT] \
  --update-env-vars DB_USER=[MY_DB_USER] \
  --update-env-vars DB_PASS=[MY_DB_PASS] \
  --update-env-vars DB_NAME=[MY_DB]
```

Replace environment variables with the correct values for your AlloyDB instance configuration.

Take note of the URL output at the end of the deployment process.

It is recommended to use the [Secret Manager integration](https://cloud.google.com/run/docs/configuring/secrets) for Cloud Run instead
of using environment variables for the SQL configuration. The service injects the AlloyDB credentials from
Secret Manager at runtime via an environment variable.

Create secrets via the command line:
```sh
echo -n $DB_USER | \
    gcloud secrets versions add DB_USER_SECRET --data-file=-
```

Deploy the service to Cloud Run specifying the env var name and secret name:
```sh
gcloud beta run deploy SERVICE --image gcr.io/[YOUR_PROJECT_ID]/alloydb \
    --update-secrets DB_HOST=[DB_HOST_SECRET]:latest,\
      DB_PORT=[DB_PORT_SECRET]:latest, \
      DB_USER=[DB_USER_SECRET]:latest, \
      DB_PASS=[DB_PASS_SECRET]:latest, \
      DB_NAME=[DB_NAME_SECRET]:latest
```

1. Navigate your browser to the URL noted in step 2.

For more details about using Cloud Run see http://cloud.run.
