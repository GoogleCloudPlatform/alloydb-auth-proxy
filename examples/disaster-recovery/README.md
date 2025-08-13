# Coordinate disaster recovery with Secret Manager

## Background

The Auth Proxy doesn't currently have any support for automatically
following a [switchover operation][switchover].

This document shows a workaround where you wrap the Auth Proxy in a
script that polls a secret in Secret Manager. The secret holds the
instance URI of your active primary instance. After a switchover,
you manually update the secret to your new active primary instance
URI. When that secret changes the Auth Proxy will automatically
restart, minimizing the number of manual steps you need to take 
to complete a switchover operation.

[switchover]: https://cloud.google.com/alloydb/docs/cross-region-replication/work-with-cross-region-replication#switchover-secondary

## Restart Auth Proxy when secret changes

Here is the wrapper script:

```sh
#! /bin/bash

SECRET_ID="my-secret-id" # TODO(developer): replace this value
REFRESH_INTERVAL=5

# Get the latest version of the secret and start the Proxy
INSTANCE_URI=$(gcloud secrets versions access "latest" --secret="$SECRET_ID")
alloydb-auth-proxy --port "$PORT" "$INSTANCE_URI" &
PID=$!

# Every 5s, get the latest version of the secret. If it's changed, restart the
# Proxy with the new value.
while true; do
    sleep $REFRESH_INTERVAL
    NEW=$(gcloud secrets versions access "latest" --secret="$SECRET_ID")
    if [ "$INSTANCE" != "$NEW" ]; then
        INSTANCE=$NEW
        kill $PID
        wait $PID
        alloydb-auth-proxy --port "$PORT" "$INSTANCE" &
        PID=$!
    fi
done
```

## Benefits of this approach

Using this approach will help assist with switchovers without needing to
reconfigure your application. Instead, by changing only how the Proxy is started,
you won't have to redeploy your application which will always connect to 127.0.0.1.
