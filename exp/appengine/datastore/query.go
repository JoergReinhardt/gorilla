// Copyright 2011 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package datastore

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"reflect"
	"strings"

	"appengine"
	"goprotobuf.googlecode.com/hg/proto"

	pb "appengine_internal/datastore"
)

// TODO
// ====
// - Ancestor Queries:
//   - http://code.google.com/appengine/docs/python/datastore/queries.html#Ancestor_Queries
// - Kindless Queries:
//   - http://code.google.com/appengine/docs/python/datastore/queries.html#Kindless_Queries
//   - http://code.google.com/appengine/docs/java/datastore/queries.html#Kindless_Queries
// - Kindless Ancestor Queries:
//   - http://code.google.com/appengine/docs/python/datastore/queries.html#Kindless_Ancestor_Queries
// - Maybe split Query.getProto() in smaller functions to perform checkings
//   related to datastore restrictions:
//   - http://code.google.com/appengine/docs/python/datastore/queries.html#Restrictions_on_Queries
// - Async calls:
//   - http://code.google.com/appengine/docs/python/datastore/async.html
// - IN, OR and != filters

// ----------------------------------------------------------------------------
// Query
// ----------------------------------------------------------------------------

// NewQuery creates a new Query for a specific entity kind.
func NewQuery(kind string) *Query {
	return &Query{kind: kind}
}

// Query represents a datastore query, and is immutable.
type Query struct {
	kind     string
	ancestor *Key
	filter   []queryFilter
	order    []queryOrder
}

// Kind sets the entity kind for the Query.
func (q *Query) Kind(kind string) *Query {
	c := *q
	c.kind = kind
	return &c
}

// Ancestor sets the ancestor filter for the Query.
func (q *Query) Ancestor(ancestor *Key) *Query {
	c := *q
	c.ancestor = ancestor
	return &c
}

// Filter adds a field-based filter to the Query.
// The filterStr argument must be a field name followed by optional space,
// followed by an operator, one of ">", "<", ">=", "<=", or "=".
// Fields are compared against the provided value using the operator.
// Multiple filters are AND'ed together.
func (q *Query) Filter(filterStr string, value interface{}) *Query {
	c := *q
	c.filter = append(c.filter, queryFilter{filterStr, value})
	return &c
}

// Order adds a field-based sort to the query.
// Orders are applied in the order they are added.
// The default order is ascending; to sort in descending
// order prefix the fieldName with a minus sign (-).
func (q *Query) Order(order string) *Query {
	c := *q
	c.order = append(c.order, queryOrder(order))
	return &c
}

// String returns a string representation of the query.
func (q *Query) String() string {
	var hasWhere bool
	buf := bytes.NewBufferString("SELECT *")
	if q.kind != "" {
		fmt.Fprintf(buf, " FROM %v", q.kind)
	}
	if q.ancestor != nil {
		fmt.Fprintf(buf, " WHERE ANCESTOR IS KEY('%v')", q.ancestor.Encode())
		hasWhere = true
	}
	if q.filter != nil {
		for _, filter := range q.filter {
			if !hasWhere {
				buf.WriteString(" WHERE")
				hasWhere = true
			} else {
				buf.WriteString(" AND")
			}
			fmt.Fprintf(buf, " %v", filter.String())
		}
	}
	if q.order != nil {
		buf.WriteString(" ORDER BY")
		for i, order := range q.order {
			if i > 0 {
				buf.WriteString(",")
			}
			fmt.Fprintf(buf, " %v", order.String())
		}
	}
	return buf.String()
}

// Run ------------------------------------------------------------------------

// Run runs the query in the given context.
func (q *Query) Run(c appengine.Context, o *QueryOptions) *Iterator {
	if o == nil {
		o = &QueryOptions{}
	}
	return newIterator(c, q, o, "RunQuery")
}

// Private methods ------------------------------------------------------------

// toProto converts the query to a protocol buffer.
//
// Values are stored as defined by the user and validation only happens here.
// It returns an ErrMulti with all encountered errors, if any.
func (q *Query) toProto(dst *pb.Query) os.Error {
	var errMulti ErrMulti
	if q.kind != "" {
		dst.Kind = proto.String(q.kind)
	} else {
		// TODO: kindless queries.
		errMulti = append(errMulti, os.NewError("datastore: empty query kind"))
	}
	if q.ancestor != nil {
		dst.Ancestor = q.ancestor.toProto()
	}
	if q.filter != nil {
		dst.Filter = make([]*pb.Query_Filter, len(q.filter))
		for i, f := range q.filter {
			var filter pb.Query_Filter
			if err := f.toProto(&filter); err != nil {
				errMulti = append(errMulti, err)
			}
			dst.Filter[i] = &filter
		}
	}
	if q.order != nil {
		dst.Order = make([]*pb.Query_Order, len(q.order))
		for i, o := range q.order {
			var order pb.Query_Order
			if err := o.toProto(&order); err != nil {
				errMulti = append(errMulti, err)
			}
			dst.Order[i] = &order
		}
	}
	if len(errMulti) > 0 {
		return errMulti
	}
	return nil
}

// ----------------------------------------------------------------------------
// QueryOptions
// ----------------------------------------------------------------------------

// NewQueryOptions creates a new configuration to run a query.
func NewQueryOptions(limit int, offset int) *QueryOptions {
	return &QueryOptions{limit: limit, offset: offset}
}

// QueryOptions defines a configuration to run a query, and is immutable.
type QueryOptions struct {
	limit       int
	offset      int
	keysOnly    bool
	compile     bool
	startCursor *Cursor
	endCursor   *Cursor
	// TODO?
	// batchSize: int, hint for the number of results returned per RPC
	// prefetchSize: int, hint for the number of results in the first RPC
}

// Limit sets the maximum number of keys/entities to return.
// A zero value means unlimited. A negative value is invalid.
func (o *QueryOptions) Limit(limit int) *QueryOptions {
	c := *o
	c.limit = limit
	return &c
}

// Offset sets how many keys to skip over before returning results.
// A negative value is invalid.
func (o *QueryOptions) Offset(offset int) *QueryOptions {
	c := *o
	c.offset = offset
	return &c
}

// KeysOnly configures the query to return keys, instead of keys and entities.
func (o *QueryOptions) KeysOnly(keysOnly bool) *QueryOptions {
	c := *o
	c.keysOnly = keysOnly
	return &c
}

// Compile configures the query to produce cursors.
func (o *QueryOptions) Compile(compile bool) *QueryOptions {
	c := *o
	c.compile = compile
	return &c
}

// Cursor sets the cursor position to start the query.
func (o *QueryOptions) Cursor(cursor *Cursor) *QueryOptions {
	// TODO: When a cursor is set, should we automatically configure it
	// to produce cursors?
	c := *o
	c.startCursor = cursor
	return &c
}

// Private methods ------------------------------------------------------------

// toProto converts the query to a protocol buffer.
//
// Values are stored as defined by the user and validation only happens here.
// It returns an ErrMulti with all encountered errors, if any.
//
// TODO: zero limit policy
func (o *QueryOptions) toProto(dst *pb.Query) os.Error {
	var errMulti ErrMulti
	if err := validInt32(o.limit, "limit"); err != nil {
		errMulti = append(errMulti, err)
	} else {
		dst.Limit = proto.Int32(int32(o.limit))
	}
	if err := validInt32(o.offset, "offset"); err != nil {
		errMulti = append(errMulti, err)
	} else {
		dst.Offset = proto.Int32(int32(o.offset))
	}
	dst.KeysOnly = proto.Bool(o.keysOnly)
	dst.Compile = proto.Bool(o.compile)
	if o.startCursor != nil {
		dst.CompiledCursor = o.startCursor.compiledCursor
	}
	if o.endCursor != nil {
		dst.EndCompiledCursor = o.endCursor.compiledCursor
	}
	if len(errMulti) > 0 {
		return errMulti
	}
	return nil
}

// ----------------------------------------------------------------------------
// queryFilter
// ----------------------------------------------------------------------------

var operatorToProto = map[string]*pb.Query_Filter_Operator{
	"<":  pb.NewQuery_Filter_Operator(pb.Query_Filter_LESS_THAN),
	"<=": pb.NewQuery_Filter_Operator(pb.Query_Filter_LESS_THAN_OR_EQUAL),
	"=":  pb.NewQuery_Filter_Operator(pb.Query_Filter_EQUAL),
	">=": pb.NewQuery_Filter_Operator(pb.Query_Filter_GREATER_THAN_OR_EQUAL),
	">":  pb.NewQuery_Filter_Operator(pb.Query_Filter_GREATER_THAN),
}

// queryFilter stores a query filter as defined by the user.
type queryFilter struct {
	filter string
	value  interface{}
}

// String returns a string representation of the filter.
func (q queryFilter) String() string {
	// TODO: value doesn't follow GQL strictly.
	return fmt.Sprintf("%v%#v", q.filter, q.value)
}

// toProto converts the filter to a pb.Query_Filter.
func (q queryFilter) toProto(dst *pb.Query_Filter) os.Error {
	property, operator, err := q.parse()
	if err != nil {
		return err
	}
	dst.Op = operatorToProto[operator]
	if dst.Op == nil {
		return fmt.Errorf("datastore: invalid operator %q in filter %q",
			operator, q.filter)
	}
	prop, errStr := valueToProto(property, reflect.ValueOf(q.value), false)
	if errStr != "" {
		return fmt.Errorf("datastore: bad query filter value type: %q", errStr)
	}
	dst.Property = []*pb.Property{prop}
	return nil
}

// parse parses the filter an returns (property, operator, err).
func (q queryFilter) parse() (property, operator string, err os.Error) {
	filter := strings.TrimSpace(q.filter)
	if filter == "" {
		err = os.NewError("datastore: invalid query filter: " + filter)
		return
	}
	property = strings.TrimRight(filter, " ><=")
	if property == "" {
		err = os.NewError("datastore: empty query filter property")
		return
	}
	operator = strings.TrimSpace(filter[len(property):])
	return
}

// ----------------------------------------------------------------------------
// queryOrder
// ----------------------------------------------------------------------------

var orderDirectionToProto = map[string]*pb.Query_Order_Direction{
	"+": pb.NewQuery_Order_Direction(pb.Query_Order_ASCENDING),
	"-": pb.NewQuery_Order_Direction(pb.Query_Order_DESCENDING),
}

// queryOrder stores a query order as defined by the user.
type queryOrder string

// String returns a string representation of the order.
func (q queryOrder) String() string {
	property, direction, _ := q.parse()
	if direction == "+" {
		direction = "ASC"
	} else {
		direction = "DESC"
	}
	return fmt.Sprintf("%v %v", property, direction)
}

// toProto converts the order to a pb.Query_Order.
func (q queryOrder) toProto(dst *pb.Query_Order) os.Error {
	property, direction, _ := q.parse()
	if property == "" {
		return os.NewError("datastore: empty query order property")
	}
	dst.Property = proto.String(property)
	dst.Direction = orderDirectionToProto[direction]
	return nil
}

// parse parses the order an returns (property, direction, err).
func (q queryOrder) parse() (property, direction string, err os.Error) {
	property = strings.TrimSpace(string(q))
	direction = "+"
	if strings.HasPrefix(property, "-") {
		property = strings.TrimSpace(property[1:])
		direction = "-"
	}
	return
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// validInt32 validates that an int is positive ad doesn't overflow.
func validInt32(value int, name string) os.Error {
	if value < 0 {
		return os.NewError("datastore: negative value for " + name)
	} else if value > math.MaxInt32 {
		return os.NewError("datastore: value overflow for " + name)
	}
	return nil
}
