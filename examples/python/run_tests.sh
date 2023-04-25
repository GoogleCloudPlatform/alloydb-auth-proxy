#! /bin/bash
# Copyright 2022 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# `-e` enables the script to automatically fail when a command fails
set -ex

# download the proxy and run it in the background listening on 127.0.0.1:5432
URL="https://storage.googleapis.com/alloydb-auth-proxy/v0.6.2"
wget "$URL/alloydb-auth-proxy.linux.amd64" -O alloydb-auth-proxy
chmod +x alloydb-auth-proxy
./alloydb-auth-proxy "${ALLOYDB_CONNECTION_NAME}" &
export INSTANCE_HOST="127.0.0.1"
export DB_PORT=5432
ps
PROXY_PID="$(pgrep alloydb)"
trap 'kill ${PROXY_PID}' 1 2 3 6 15

cd examples/python

pip install -r requirements.txt
pip install -r requirements-test.txt

# log python version info
echo "Running tests using Python:"
python --version
python -m pytest --version

python -m pytest --tb=long .
