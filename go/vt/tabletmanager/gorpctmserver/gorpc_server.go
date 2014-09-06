// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gorpctmserver

import (
	"fmt"
	"time"

	"code.google.com/p/go.net/context"
	mproto "github.com/youtube/vitess/go/mysql/proto"
	"github.com/youtube/vitess/go/rpcwrap"
	blproto "github.com/youtube/vitess/go/vt/binlog/proto"
	myproto "github.com/youtube/vitess/go/vt/mysqlctl/proto"
	"github.com/youtube/vitess/go/vt/rpc"
	"github.com/youtube/vitess/go/vt/tabletmanager"
	"github.com/youtube/vitess/go/vt/tabletmanager/actionnode"
	"github.com/youtube/vitess/go/vt/tabletmanager/actor"
	"github.com/youtube/vitess/go/vt/tabletmanager/gorpcproto"
	"github.com/youtube/vitess/go/vt/topo"
	"github.com/youtube/vitess/go/vt/topotools"
)

// TabletManager is the Go RPC implementation of the RPC service
type TabletManager struct {
	agent *tabletmanager.ActionAgent
}

//
// Various read-only methods
//

func (tm *TabletManager) Ping(ctx context.Context, args, reply *string) error {
	return tm.agent.RpcWrap(ctx, actionnode.TABLET_ACTION_PING, args, reply, func() error {
		*reply = *args
		return nil
	})
}

func (tm *TabletManager) GetSchema(ctx context.Context, args *gorpcproto.GetSchemaArgs, reply *myproto.SchemaDefinition) error {
	return tm.agent.RpcWrap(ctx, actionnode.TABLET_ACTION_GET_SCHEMA, args, reply, func() error {
		// read the tablet to get the dbname
		tablet, err := tm.agent.TopoServer.GetTablet(tm.agent.TabletAlias)
		if err != nil {
			return err
		}

		// and get the schema
		sd, err := tm.agent.Mysqld.GetSchema(tablet.DbName(), args.Tables, args.ExcludeTables, args.IncludeViews)
		if err == nil {
			*reply = *sd
		}
		return err
	})
}

func (tm *TabletManager) GetPermissions(ctx context.Context, args *rpc.UnusedRequest, reply *myproto.Permissions) error {
	return tm.agent.RpcWrap(ctx, actionnode.TABLET_ACTION_GET_PERMISSIONS, args, reply, func() error {
		p, err := tm.agent.Mysqld.GetPermissions()
		if err == nil {
			*reply = *p
		}
		return err
	})
}

//
// Various read-write methods
//

func (tm *TabletManager) ChangeType(ctx context.Context, args *topo.TabletType, reply *rpc.UnusedResponse) error {
	return tm.agent.RpcWrapLockAction(ctx, actionnode.TABLET_ACTION_CHANGE_TYPE, args, reply, func() error {
		return topotools.ChangeType(tm.agent.TopoServer, tm.agent.TabletAlias, *args, nil, true /*runHooks*/)
	})
}

func (tm *TabletManager) SetBlacklistedTables(ctx context.Context, args *gorpcproto.SetBlacklistedTablesArgs, reply *rpc.UnusedResponse) error {
	return tm.agent.RpcWrapLockAction(ctx, actionnode.TABLET_ACTION_SET_BLACKLISTED_TABLES, args, reply, func() error {
		return actor.SetBlacklistedTables(tm.agent.TopoServer, tm.agent.TabletAlias, args.Tables)
	})
}

func (tm *TabletManager) ReloadSchema(ctx context.Context, args *rpc.UnusedRequest, reply *rpc.UnusedResponse) error {
	return tm.agent.RpcWrapLockActionSchema(ctx, actionnode.TABLET_ACTION_RELOAD_SCHEMA, args, reply, func() error {
		// no-op, the framework will force the schema reload
		return nil
	})
}

func (tm *TabletManager) ExecuteFetch(ctx context.Context, args *gorpcproto.ExecuteFetchArgs, reply *mproto.QueryResult) error {
	return tm.agent.RpcWrap(ctx, actionnode.TABLET_ACTION_EXECUTE_FETCH, args, reply, func() error {
		qr, err := tm.agent.ExecuteFetch(args.Query, args.MaxRows, args.WantFields, args.DisableBinlogs)
		if err == nil {
			*reply = *qr
		}
		return err
	})
}

//
// Replication related methods
//

func (tm *TabletManager) SlaveStatus(ctx context.Context, args *rpc.UnusedRequest, reply *myproto.ReplicationStatus) error {
	return tm.agent.RpcWrap(ctx, actionnode.TABLET_ACTION_SLAVE_STATUS, args, reply, func() error {
		status, err := tm.agent.Mysqld.SlaveStatus()
		if err == nil {
			*reply = *status
		}
		return err
	})
}

func (tm *TabletManager) WaitSlavePosition(ctx context.Context, args *gorpcproto.WaitSlavePositionArgs, reply *myproto.ReplicationStatus) error {
	return tm.agent.RpcWrap(ctx, actionnode.TABLET_ACTION_WAIT_SLAVE_POSITION, args, reply, func() error {
		if err := tm.agent.Mysqld.WaitMasterPos(args.Position, args.WaitTimeout); err != nil {
			return err
		}

		status, err := tm.agent.Mysqld.SlaveStatus()
		if err == nil {
			*reply = *status
		}
		return err
	})
}

func (tm *TabletManager) MasterPosition(ctx context.Context, args *rpc.UnusedRequest, reply *myproto.ReplicationPosition) error {
	return tm.agent.RpcWrap(ctx, actionnode.TABLET_ACTION_MASTER_POSITION, args, reply, func() error {
		position, err := tm.agent.Mysqld.MasterPosition()
		if err == nil {
			*reply = position
		}
		return err
	})
}

func (tm *TabletManager) StopSlave(ctx context.Context, args *rpc.UnusedRequest, reply *rpc.UnusedResponse) error {
	return tm.agent.RpcWrapLock(ctx, actionnode.TABLET_ACTION_STOP_SLAVE, args, reply, func() error {
		return tm.agent.Mysqld.StopSlave(map[string]string{"TABLET_ALIAS": tm.agent.TabletAlias.String()})
	})
}

func (tm *TabletManager) StopSlaveMinimum(ctx context.Context, args *gorpcproto.StopSlaveMinimumArgs, reply *myproto.ReplicationStatus) error {
	return tm.agent.RpcWrapLock(ctx, actionnode.TABLET_ACTION_STOP_SLAVE_MINIMUM, args, reply, func() error {
		if err := tm.agent.Mysqld.WaitMasterPos(args.Position, args.WaitTime); err != nil {
			return err
		}
		if err := tm.agent.Mysqld.StopSlave(map[string]string{"TABLET_ALIAS": tm.agent.TabletAlias.String()}); err != nil {
			return err
		}
		status, err := tm.agent.Mysqld.SlaveStatus()
		if err == nil {
			*reply = *status
		}
		return err
	})
}

func (tm *TabletManager) StartSlave(ctx context.Context, args *rpc.UnusedRequest, reply *rpc.UnusedResponse) error {
	return tm.agent.RpcWrapLock(ctx, actionnode.TABLET_ACTION_START_SLAVE, args, reply, func() error {
		return tm.agent.Mysqld.StartSlave(map[string]string{"TABLET_ALIAS": tm.agent.TabletAlias.String()})
	})
}

func (tm *TabletManager) TabletExternallyReparented(ctx context.Context, args *rpc.UnusedRequest, reply *rpc.UnusedResponse) error {
	// TODO(alainjobart) we should forward the RPC deadline from
	// the original gorpc call. Until we support that, use a
	// reasonnable hard-coded value.
	return tm.agent.RpcWrapLockAction(ctx, actionnode.TABLET_ACTION_EXTERNALLY_REPARENTED, args, reply, func() error {
		return actor.TabletExternallyReparented(tm.agent.TopoServer, tm.agent.TabletAlias, 30*time.Second, *tabletmanager.LockTimeout)
	})
}

func (tm *TabletManager) GetSlaves(ctx context.Context, args *rpc.UnusedRequest, reply *gorpcproto.GetSlavesReply) error {
	return tm.agent.RpcWrap(ctx, actionnode.TABLET_ACTION_GET_SLAVES, args, reply, func() error {
		var err error
		reply.Addrs, err = tm.agent.Mysqld.FindSlaves()
		return err
	})
}

func (tm *TabletManager) WaitBlpPosition(ctx context.Context, args *gorpcproto.WaitBlpPositionArgs, reply *rpc.UnusedResponse) error {
	return tm.agent.RpcWrap(ctx, actionnode.TABLET_ACTION_WAIT_BLP_POSITION, args, reply, func() error {
		return tm.agent.Mysqld.WaitBlpPos(&args.BlpPosition, args.WaitTimeout)
	})
}

func (tm *TabletManager) StopBlp(ctx context.Context, args *rpc.UnusedRequest, reply *blproto.BlpPositionList) error {
	return tm.agent.RpcWrapLock(ctx, actionnode.TABLET_ACTION_STOP_BLP, args, reply, func() error {
		if tm.agent.BinlogPlayerMap == nil {
			return fmt.Errorf("No BinlogPlayerMap configured")
		}
		tm.agent.BinlogPlayerMap.Stop()
		positions, err := tm.agent.BinlogPlayerMap.BlpPositionList()
		if err != nil {
			return err
		}
		*reply = *positions
		return nil
	})
}

func (tm *TabletManager) StartBlp(ctx context.Context, args *rpc.UnusedRequest, reply *rpc.UnusedResponse) error {
	return tm.agent.RpcWrapLock(ctx, actionnode.TABLET_ACTION_START_BLP, args, reply, func() error {
		if tm.agent.BinlogPlayerMap == nil {
			return fmt.Errorf("No BinlogPlayerMap configured")
		}
		tm.agent.BinlogPlayerMap.Start()
		return nil
	})
}

func (tm *TabletManager) RunBlpUntil(ctx context.Context, args *gorpcproto.RunBlpUntilArgs, reply *myproto.ReplicationPosition) error {
	return tm.agent.RpcWrapLock(ctx, actionnode.TABLET_ACTION_RUN_BLP_UNTIL, args, reply, func() error {
		if tm.agent.BinlogPlayerMap == nil {
			return fmt.Errorf("No BinlogPlayerMap configured")
		}
		if err := tm.agent.BinlogPlayerMap.RunUntil(args.BlpPositionList, args.WaitTimeout); err != nil {
			return err
		}
		position, err := tm.agent.Mysqld.MasterPosition()
		if err == nil {
			*reply = position
		}
		return err
	})
}

//
// Reparenting related functions
//

func (tm *TabletManager) SlaveWasPromoted(ctx context.Context, args *rpc.UnusedRequest, reply *rpc.UnusedResponse) error {
	return tm.agent.RpcWrapLockAction(ctx, actionnode.TABLET_ACTION_SLAVE_WAS_PROMOTED, args, reply, func() error {
		return actor.SlaveWasPromoted(tm.agent.TopoServer, tm.agent.TabletAlias)
	})
}

func (tm *TabletManager) SlaveWasRestarted(ctx context.Context, args *actionnode.SlaveWasRestartedArgs, reply *rpc.UnusedResponse) error {
	return tm.agent.RpcWrapLockAction(ctx, actionnode.TABLET_ACTION_SLAVE_WAS_RESTARTED, args, reply, func() error {
		return actor.SlaveWasRestarted(tm.agent.TopoServer, tm.agent.TabletAlias, args)
	})
}

// registration glue

func init() {
	tabletmanager.RegisterQueryServices = append(tabletmanager.RegisterQueryServices, func(agent *tabletmanager.ActionAgent) {
		rpcwrap.RegisterAuthenticated(&TabletManager{agent})
	})
}
