// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package alloydb

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"fmt"
	"regexp"
	"sync"
	"time"

	"cloud.google.com/go/alloydbconn/errtype"
	"cloud.google.com/go/alloydbconn/internal/alloydbapi"
)

const (
	// refreshBuffer is the amount of time before a result expires to start a
	// new refresh attempt.
	refreshBuffer = 12 * time.Hour
)

var (
	// Instance URI is in the format:
	// '/projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>'
	// Additionally, we have to support legacy "domain-scoped" projects (e.g. "google.com:PROJECT")
	instURIRegex = regexp.MustCompile("projects/([^:]+(:[^:]+)?)/locations/([^:]+)/clusters/([^:]+)/instances/([^:]+)")
)

// instanceURI reprents an AlloyDB instance.
type instanceURI struct {
	project string
	region  string
	cluster string
	name    string
}

func (i *instanceURI) String() string {
	return fmt.Sprintf("%s/%s/%s/%s", i.project, i.region, i.cluster, i.name)
}

// parseInstURI initializes a new instanceURI struct.
func parseInstURI(cn string) (instanceURI, error) {
	b := []byte(cn)
	m := instURIRegex.FindSubmatch(b)
	if m == nil {
		err := errtype.NewConfigError(
			"invalid instance URI, expected projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>",
			cn,
		)
		return instanceURI{}, err
	}

	c := instanceURI{
		project: string(m[1]),
		region:  string(m[3]),
		cluster: string(m[4]),
		name:    string(m[5]),
	}
	return c, nil
}

type metadata struct {
	ipAddrs map[string]string
	version string
}

// refreshOperation is a pending result of a refresh operation of data used to connect securely. It should
// only be initialized by the Instance struct as part of a refresh cycle.
type refreshOperation struct {
	result refreshResult
	err    error

	// timer that triggers refresh, can be used to cancel.
	timer *time.Timer
	// indicates the struct is ready to read from
	ready chan struct{}
}

// Cancel prevents the instanceInfo from starting, if it hasn't already started. Returns true if timer
// was stopped successfully, or false if it has already started.
func (r *refreshOperation) Cancel() bool {
	return r.timer.Stop()
}

// Wait blocks until the refreshOperation attempt is completed.
func (r *refreshOperation) Wait(ctx context.Context) error {
	select {
	case <-r.ready:
		return r.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// IsValid returns true if this result is complete, successful, and is still valid.
func (r *refreshOperation) IsValid() bool {
	// verify the result has finished running
	select {
	default:
		return false
	case <-r.ready:
		if r.err != nil || time.Now().After(r.result.expiry) {
			return false
		}
		return true
	}
}

// Instance manages the information used to connect to the AlloyDB instance by
// periodically calling the AlloyDB Admin API. It automatically refreshes the
// required information approximately 5 minutes before the previous certificate
// expires (every 55 minutes).
type Instance struct {
	instanceURI
	key *rsa.PrivateKey
	r   refresher

	resultGuard sync.RWMutex
	// cur represents the current refreshOperation that will be used to create connections. If a valid complete
	// refreshOperation isn't available it's possible for cur to be equal to next.
	cur *refreshOperation
	// next represents a future or ongoing refreshOperation. Once complete, it will replace cur and schedule a
	// replacement to occur.
	next *refreshOperation

	// OpenConns is the number of open connections to the instance.
	OpenConns uint64

	// ctx is the default ctx for refresh operations. Canceling it prevents new refresh
	// operations from being triggered.
	ctx    context.Context
	cancel context.CancelFunc
}

// NewInstance initializes a new Instance given an instance URI
func NewInstance(
	instance string,
	client *alloydbapi.Client,
	key *rsa.PrivateKey,
	refreshTimeout time.Duration,
	dialerID string,
) (*Instance, error) {
	cn, err := parseInstURI(instance)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	i := &Instance{
		instanceURI: cn,
		key:         key,
		r: newRefresher(
			client,
			refreshTimeout,
			30*time.Second,
			2,
			dialerID,
		),
		ctx:    ctx,
		cancel: cancel,
	}
	// For the initial refresh operation, set cur = next so that connection requests block
	// until the first refresh is complete.
	i.resultGuard.Lock()
	i.cur = i.scheduleRefresh(0)
	i.next = i.cur
	i.resultGuard.Unlock()
	return i, nil
}

// Close closes the instance; it stops the refresh cycle and prevents it from
// making additional calls to the AlloyDB Admin API.
func (i *Instance) Close() {
	i.cancel()
}

// ConnectInfo returns an IP address of the AlloyDB instance.
func (i *Instance) ConnectInfo(ctx context.Context) (string, *tls.Config, error) {
	res, err := i.result(ctx)
	if err != nil {
		return "", nil, err
	}
	return res.result.instanceIPAddr, res.result.conf, nil
}

// ForceRefresh triggers an immediate refresh operation to be scheduled and used for future connection attempts.
func (i *Instance) ForceRefresh() {
	i.resultGuard.Lock()
	defer i.resultGuard.Unlock()
	// If the next refresh hasn't started yet, we can cancel it and start an immediate one
	if i.next.Cancel() {
		i.next = i.scheduleRefresh(0)
	}
	// block all sequential connection attempts on the next refresh result
	i.cur = i.next
}

// result returns the most recent refresh result (waiting for it to complete if necessary)
func (i *Instance) result(ctx context.Context) (*refreshOperation, error) {
	i.resultGuard.RLock()
	res := i.cur
	i.resultGuard.RUnlock()
	err := res.Wait(ctx)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// scheduleRefresh schedules a refresh operation to be triggered after a given
// duration. The returned refreshOperation can be used to either Cancel or Wait
// for the operations result.
func (i *Instance) scheduleRefresh(d time.Duration) *refreshOperation {
	res := &refreshOperation{}
	res.ready = make(chan struct{})
	res.timer = time.AfterFunc(d, func() {
		res.result, res.err = i.r.performRefresh(i.ctx, i.instanceURI, i.key)
		close(res.ready)

		// Once the refresh is complete, update "current" with working result and schedule a new refresh
		i.resultGuard.Lock()
		defer i.resultGuard.Unlock()
		// if failed, scheduled the next refresh immediately
		if res.err != nil {
			i.next = i.scheduleRefresh(0)
			// If the latest result is bad, avoid replacing the used result while it's
			// still valid and potentially able to provide successful connections.
			// TODO: This means that errors while the current result is still valid are
			// surpressed. We should try to surface errors in a more meaningful way.
			if !i.cur.IsValid() {
				i.cur = res
			}
			return
		}
		// Update the current results, and schedule the next refresh in the future
		i.cur = res
		select {
		case <-i.ctx.Done():
			// instance has been closed, don't schedule anything
			return
		default:
		}
		nextRefresh := i.cur.result.expiry.Add(-refreshBuffer)
		i.next = i.scheduleRefresh(time.Until(nextRefresh))
	})
	return res
}

// String returns the instance's URI.
func (i *Instance) String() string {
	return i.instanceURI.String()
}
