// Copyright 2014, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package planbuilder

import (
	"fmt"

	"github.com/youtube/vitess/go/vt/sqlparser"
)

type PlanID int

const (
	NoPlan = PlanID(iota)
	SelectUnsharded
	SelectSingleShardKey
	SelectMultiShardKey
	SelectSingleLookup
	SelectMultiLookup
	SelectScatter
	UpdateUnsharded
	UpdateSingleShardKey
	UpdateSingleLookup
	DeleteUnsharded
	DeleteSingleShardKey
	DeleteSingleLookup
	InsertUnsharded
	InsertSharded
	NumPlans
)

type Plan struct {
	ID        PlanID
	Reason    string
	Table     *Table
	Original  string
	Rewritten string
	Index     *Index
	Values    interface{}
}

func (pln *Plan) Size() int {
	return 1
}

// Must exactly match order of plan constants.
var planName = []string{
	"NoPlan",
	"SelectUnsharded",
	"SelectSingleShardKey",
	"SelectMultiShardKey",
	"SelectSingleLookup",
	"SelectMultiLookup",
	"SelectScatter",
	"UpdateUnsharded",
	"UpdateSingleShardKey",
	"UpdateSingleLookup",
	"DeleteUnsharded",
	"DeleteSingleShardKey",
	"DeleteSingleLookup",
	"InsertUnsharded",
	"InsertSharded",
}

func (id PlanID) String() string {
	if id < 0 || id >= NumPlans {
		return ""
	}
	return planName[id]
}

func PlanByName(s string) (id PlanID, ok bool) {
	for i, v := range planName {
		if v == s {
			return PlanID(i), true
		}
	}
	return NumPlans, false
}

func (id PlanID) IsMulti() bool {
	return id == SelectMultiShardKey || id == SelectMultiLookup || id == SelectScatter
}

func (id PlanID) MarshalJSON() ([]byte, error) {
	return ([]byte)(fmt.Sprintf("\"%s\"", id.String())), nil
}

func BuildPlan(query string, schema *Schema) *Plan {
	statement, err := sqlparser.Parse(query)
	if err != nil {
		return &Plan{
			ID:       NoPlan,
			Reason:   err.Error(),
			Original: query,
		}
	}
	noplan := &Plan{
		ID:       NoPlan,
		Reason:   "too complex",
		Original: query,
	}
	var plan *Plan
	switch statement := statement.(type) {
	case *sqlparser.Select:
		plan = buildSelectPlan(statement, schema)
	case *sqlparser.Insert:
		plan = buildInsertPlan(statement, schema)
	case *sqlparser.Update:
		plan = buildUpdatePlan(statement, schema)
	case *sqlparser.Delete:
		plan = buildDeletePlan(statement, schema)
	case *sqlparser.Union, *sqlparser.Set, *sqlparser.DDL, *sqlparser.Other:
		return noplan
	default:
		panic("unexpected")
	}
	plan.Original = query
	return plan
}

func generateQuery(statement sqlparser.Statement) string {
	buf := sqlparser.NewTrackedBuffer(nil)
	statement.Format(buf)
	return buf.String()
}
