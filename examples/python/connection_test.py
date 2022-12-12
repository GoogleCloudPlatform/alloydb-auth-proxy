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

import logging

from flask.testing import FlaskClient

import pytest

import app

logger = logging.getLogger()


@pytest.fixture(scope="module")
def client() -> FlaskClient:
    app.app.testing = True
    client = app.app.test_client()
    return client


def test_get_votes(client: FlaskClient) -> None:
    response = client.get("/")
    text = "Tabs VS Spaces"
    body = response.text
    assert response.status_code == 200
    assert text in body


def test_cast_vote(client: FlaskClient) -> None:
    response = client.post("/votes", data={"team": "SPACES"})
    text = "Vote successfully cast for 'SPACES'"
    body = response.text
    assert response.status_code == 200
    assert text in body
