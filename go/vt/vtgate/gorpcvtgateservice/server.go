// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gorpcvtgateservice provides to go rpc glue for vtgate
package gorpcvtgateservice

import (
	"code.google.com/p/go.net/context"
	"github.com/youtube/vitess/go/rpcwrap"
	"github.com/youtube/vitess/go/vt/rpc"
	"github.com/youtube/vitess/go/vt/vtgate"
	"github.com/youtube/vitess/go/vt/vtgate/proto"
)

type VTGate struct {
	server *vtgate.VTGate
}

func (vtg *VTGate) ExecuteShard(ctx context.Context, query *proto.QueryShard, reply *proto.QueryResult) error {
	return vtg.server.ExecuteShard(ctx, query, reply)
}

func (vtg *VTGate) ExecuteKeyspaceIds(ctx context.Context, query *proto.KeyspaceIdQuery, reply *proto.QueryResult) error {
	return vtg.server.ExecuteKeyspaceIds(ctx, query, reply)
}

func (vtg *VTGate) ExecuteKeyRanges(ctx context.Context, query *proto.KeyRangeQuery, reply *proto.QueryResult) error {
	return vtg.server.ExecuteKeyRanges(ctx, query, reply)
}

func (vtg *VTGate) ExecuteEntityIds(ctx context.Context, query *proto.EntityIdsQuery, reply *proto.QueryResult) error {
	return vtg.server.ExecuteEntityIds(ctx, query, reply)
}

func (vtg *VTGate) ExecuteBatchShard(ctx context.Context, batchQuery *proto.BatchQueryShard, reply *proto.QueryResultList) error {
	return vtg.server.ExecuteBatchShard(ctx, batchQuery, reply)
}

func (vtg *VTGate) ExecuteBatchKeyspaceIds(ctx context.Context, batchQuery *proto.KeyspaceIdBatchQuery, reply *proto.QueryResultList) error {
	return vtg.server.ExecuteBatchKeyspaceIds(ctx, batchQuery, reply)
}

func (vtg *VTGate) StreamExecuteShard(ctx context.Context, query *proto.QueryShard, sendReply func(interface{}) error) error {
	return vtg.server.StreamExecuteShard(ctx, query, func(value *proto.QueryResult) error {
		return sendReply(value)
	})
}

func (vtg *VTGate) StreamExecuteKeyRanges(ctx context.Context, query *proto.KeyRangeQuery, sendReply func(interface{}) error) error {
	return vtg.server.StreamExecuteKeyRanges(ctx, query, func(value *proto.QueryResult) error {
		return sendReply(value)
	})
}

func (vtg *VTGate) StreamExecuteKeyspaceIds(ctx context.Context, query *proto.KeyspaceIdQuery, sendReply func(interface{}) error) error {
	return vtg.server.StreamExecuteKeyspaceIds(ctx, query, func(value *proto.QueryResult) error {
		return sendReply(value)
	})
}

func (vtg *VTGate) Begin(ctx context.Context, noInput *rpc.UnusedRequest, outSession *proto.Session) error {
	return vtg.server.Begin(ctx, outSession)
}

func (vtg *VTGate) Commit(ctx context.Context, inSession *proto.Session, noOutput *rpc.UnusedResponse) error {
	return vtg.server.Commit(ctx, inSession)
}

func (vtg *VTGate) Rollback(ctx context.Context, inSession *proto.Session, noOutput *rpc.UnusedResponse) error {
	return vtg.server.Rollback(ctx, inSession)
}

func init() {
	vtgate.RegisterVTGates = append(vtgate.RegisterVTGates, func(vtGate *vtgate.VTGate) {
		rpcwrap.RegisterAuthenticated(&VTGate{vtGate})
	})
}
