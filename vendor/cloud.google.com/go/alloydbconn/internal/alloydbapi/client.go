// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package alloydbapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	htransport "google.golang.org/api/transport/http"
)

type InstanceGetResponse struct {
	ServerResponse googleapi.ServerResponse
	Name           string `json:"name"`
	State          string `json:"state"`
	IPAddress      string `json:"ipAddress"`
}

type GenerateClientCertificateRequest struct {
	PemCSR string `json:"pemCsr"`
}

type GenerateClientCertificateResponse struct {
	ServerResponse      googleapi.ServerResponse
	PemCertificate      string   `json:"pemCertificate"`
	PemCertificateChain []string `json:"pemCertificateChain"`
}

// baseURL is the production API endpoint of the AlloyDB Admin API
const baseURL = "https://alloydb.googleapis.com/v1beta"

type Client struct {
	client *http.Client
	// endpoint is the base URL for the AlloyDB Admin API (e.g.
	// https://alloydb.googleapis.com/v1beta)
	endpoint string
}

func NewClient(ctx context.Context, opts ...option.ClientOption) (*Client, error) {
	os := append([]option.ClientOption{
		option.WithEndpoint(baseURL),
	}, opts...) // allow for overriding the endpoint
	os = append(os,
		// do not allow for overriding the scopes
		option.WithScopes("https://www.googleapis.com/auth/cloud-platform"),
	)
	client, endpoint, err := htransport.NewClient(ctx, os...)
	if err != nil {
		return nil, err
	}
	return &Client{client: client, endpoint: endpoint}, nil
}

func (c *Client) InstanceGet(ctx context.Context, project, region, cluster, instance string) (InstanceGetResponse, error) {
	u := fmt.Sprintf(
		"%s/projects/%s/locations/%s/clusters/%s/instances/%s",
		c.endpoint, project, region, cluster, instance,
	)
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return InstanceGetResponse{}, err
	}
	res, err := c.client.Do(req)
	if err != nil {
		return InstanceGetResponse{}, err
	}
	defer res.Body.Close()

	// If the status code is 300 or greater, capture any information in the
	// response and return it as part of the error.
	if res.StatusCode >= http.StatusMultipleChoices {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return InstanceGetResponse{}, err
		}

		return InstanceGetResponse{}, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
			Body:   string(body),
		}
	}
	ret := InstanceGetResponse{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return InstanceGetResponse{}, err
	}
	return ret, nil
}

func (c *Client) GenerateClientCert(ctx context.Context, project, region, cluster string, csr []byte) (GenerateClientCertificateResponse, error) {
	u := fmt.Sprintf(
		"%s/projects/%s/locations/%s/clusters/%s:generateClientCertificate",
		c.endpoint, project, region, cluster,
	)
	body, err := json.Marshal(GenerateClientCertificateRequest{PemCSR: string(csr)})
	if err != nil {
		return GenerateClientCertificateResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(body))
	if err != nil {
		return GenerateClientCertificateResponse{}, err
	}
	res, err := c.client.Do(req.WithContext(ctx))
	if err != nil {
		return GenerateClientCertificateResponse{}, err
	}
	defer res.Body.Close()
	// If the status code is 300 or greater, capture any information in the
	// response and return it as part of the error.
	if res.StatusCode >= http.StatusMultipleChoices {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return GenerateClientCertificateResponse{}, err
		}

		return GenerateClientCertificateResponse{}, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
			Body:   string(body),
		}
	}
	ret := GenerateClientCertificateResponse{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return GenerateClientCertificateResponse{}, err
	}
	return ret, nil
}
