// Copyright 2011 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package datastore

import (
	"bytes"
	"encoding/base64"
	"gob"
	"os"
	"strconv"
	"strings"

	"appengine"
	"goprotobuf.googlecode.com/hg/proto"

	pb "appengine_internal/datastore"
)

// ----------------------------------------------------------------------------
// Key
// ----------------------------------------------------------------------------

// NewIncompleteKey creates a new incomplete key.
// kind cannot be empty.
func NewIncompleteKey(c appengine.Context, kind string, parent *Key) *Key {
	return NewKey(c, kind, "", 0, parent)
}

// NewKey creates a new key.
// kind cannot be empty.
// Either one or both of stringID and intID must be zero. If both are zero,
// the key returned is incomplete.
// parent must either be a complete key or nil.
func NewKey(c appengine.Context, kind, stringID string, intID int64, parent *Key) *Key {
	// TODO: get namespace from context once it supports it.
	return &Key{
		appID:    c.FullyQualifiedAppID(),
		parent:   parent,
		kind:     kind,
		stringID: stringID,
		intID:    intID,
	}
}

// NewNamespaceKey is the same as NewKey, but creates a key with a
// namespace.
//
// This is a temporary function to fill the gap until appengine.Context
// supports namespaces.
//
// namespace must either be a valid namespace or zero for the default
// namespace. Valid namespaces match the pattern `^[0-9A-Za-z._\-]{0,100}$`.
// If the namespace is zero, the namespace from the parent is used.
// If the namespace is not zero, it must match the namespace from the parent.
func NewNamespaceKey(c appengine.Context, kind, stringID string, intID int64, parent *Key, namespace string) *Key {
	for p := parent; p != nil && namespace == ""; p = p.parent {
		namespace = p.namespace
	}
	return &Key{
		appID:     c.FullyQualifiedAppID(),
		namespace: namespace,
		parent:    parent,
		kind:      kind,
		stringID:  stringID,
		intID:     intID,
	}
}

// Key represents the datastore key for a stored entity, and is immutable.
type Key struct {
	appID     string
	namespace string
	parent    *Key
	kind      string
	stringID  string
	intID     int64
}

// AppID returns the key's application ID.
func (k *Key) AppID() string {
	return k.appID
}

// Namespace returns the key's namespace.
func (k *Key) Namespace() string {
	return k.namespace
}

// Parent returns the key's parent key, which may be nil.
func (k *Key) Parent() *Key {
	return k.parent
}

// Kind returns the key's kind (also known as entity type).
func (k *Key) Kind() string {
	return k.kind
}

// StringID returns the key's string ID (also known as an entity name or key
// name), which may be "".
func (k *Key) StringID() string {
	return k.stringID
}

// IntID returns the key's integer ID, which may be 0.
func (k *Key) IntID() int64 {
	return k.intID
}

// Incomplete returns whether the key does not refer to a stored entity.
// In particular, whether the key has a zero StringID and a zero IntID.
func (k *Key) Incomplete() bool {
	return k.stringID == "" && k.intID == 0
}

// String returns a string representation of the key.
func (k *Key) String() string {
	if k == nil {
		return ""
	}
	b := bytes.NewBuffer(make([]byte, 0, 512))
	k.marshal(b)
	return b.String()
}

// Eq returns whether two keys are equal.
func (k *Key) Eq(o *Key) bool {
	for k != nil && o != nil {
		if k.kind != o.kind || k.stringID != o.stringID || k.intID != o.intID || k.namespace != o.namespace || k.appID != o.appID {
			return false
		}
		k, o = k.parent, o.parent
	}
	return k == o
}

// Encoding -------------------------------------------------------------------

// Encode returns an opaque representation of the key
// suitable for use in HTML and URLs.
// This is compatible with the Python and Java runtimes.
func (k *Key) Encode() string {
	ref := keyToProto(k)

	b, err := proto.Marshal(ref)
	if err != nil {
		panic(err)
	}

	// Trailing padding is stripped.
	return strings.TrimRight(base64.URLEncoding.EncodeToString(b), "=")
}

func (k *Key) GobEncode() ([]byte, os.Error) {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(keyToGobKey(k)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (k *Key) GobDecode(buf []byte) os.Error {
	gk := new(gobKey)
	if err := gob.NewDecoder(bytes.NewBuffer(buf)).Decode(gk); err != nil {
		return err
	}
	*k = *gobKeyToKey(gk)
	return nil
}

func (k *Key) MarshalJSON() ([]byte, os.Error) {
	return []byte(`"` + k.Encode() + `"`), nil
}

func (k *Key) UnmarshalJSON(buf []byte) os.Error {
	if len(buf) < 2 || buf[0] != '"' || buf[len(buf)-1] != '"' {
		return os.NewError("datastore: bad JSON key")
	}
	k2, err := DecodeKey(string(buf[1 : len(buf)-1]))
	if err != nil {
		return err
	}
	*k = *k2
	return nil
}

// Private methods ------------------------------------------------------------

// valid returns whether the key is valid.
func (k *Key) valid() bool {
	if k == nil {
		return false
	}
	for ; k != nil; k = k.parent {
		if k.kind == "" || k.appID == "" {
			return false
		}
		if k.stringID != "" && k.intID != 0 {
			return false
		}
		if k.parent != nil {
			if k.parent.Incomplete() {
				return false
			}
			if k.parent.appID != k.appID {
				return false
			}
			if k.parent.namespace != k.namespace {
				return false
			}
		}
	}
	return true
}

// root returns the furthest ancestor of a key, which may be itself.
func (k *Key) root() *Key {
	for k.parent != nil {
		k = k.parent
	}
	return k
}

// marshal marshals the key's string representation to the buffer.
func (k *Key) marshal(b *bytes.Buffer) {
	if k.parent != nil {
		k.parent.marshal(b)
	}
	b.WriteString(k.namespace)
	b.WriteByte('/')
	b.WriteString(k.kind)
	b.WriteByte(',')
	if k.stringID != "" {
		b.WriteString(k.stringID)
	} else {
		b.WriteString(strconv.Itoa64(k.intID))
	}
}

// ----------------------------------------------------------------------------
// gobKey
// ----------------------------------------------------------------------------

type gobKey struct {
	Kind      string
	StringID  string
	IntID     int64
	Parent    *gobKey
	AppID     string
	Namespace string
}

func keyToGobKey(k *Key) *gobKey {
	if k == nil {
		return nil
	}
	return &gobKey{
		Kind:      k.kind,
		StringID:  k.stringID,
		IntID:     k.intID,
		Parent:    keyToGobKey(k.parent),
		AppID:     k.appID,
		Namespace: k.namespace,
	}
}

func gobKeyToKey(gk *gobKey) *Key {
	if gk == nil {
		return nil
	}
	return &Key{
		kind:      gk.Kind,
		stringID:  gk.StringID,
		intID:     gk.IntID,
		parent:    gobKeyToKey(gk.Parent),
		appID:     gk.AppID,
		namespace: gk.Namespace,
	}
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// DecodeKey decodes a key from the opaque representation returned by Encode.
func DecodeKey(encoded string) (*Key, os.Error) {
	// Re-add padding.
	if m := len(encoded) % 4; m != 0 {
		encoded += strings.Repeat("=", 4-m)
	}

	b, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	ref := new(pb.Reference)
	if err := proto.Unmarshal(b, ref); err != nil {
		return nil, err
	}

	return protoToKey(ref)
}

// Private helpers ------------------------------------------------------------

// It's unfortunate that the two semantically equivalent concepts pb.Reference
// and pb.PropertyValue_ReferenceValue aren't the same type. For example, the
// two have different protobuf field numbers.

// protoToKey converts a Reference proto to a *Key.
func protoToKey(r *pb.Reference) (k *Key, err os.Error) {
	appID := proto.GetString(r.App)
	namespace := proto.GetString(r.NameSpace)
	for _, e := range r.Path.Element {
		k = &Key{
			appID:     appID,
			namespace: namespace,
			parent:    k,
			kind:      proto.GetString(e.Type),
			stringID:  proto.GetString(e.Name),
			intID:     proto.GetInt64(e.Id),
		}
		if !k.valid() {
			return nil, ErrInvalidKey
		}
	}
	return
}

// keyToProto converts a key to a Reference protocol buffer.
func keyToProto(k *Key) *pb.Reference {
	var namespace *string
	if k.namespace != "" {
		namespace = proto.String(k.namespace)
	}
	n := 0
	for i := k; i != nil; i = i.parent {
		n++
	}
	e := make([]*pb.Path_Element, n)
	for i := k; i != nil; i = i.parent {
		n--
		e[n] = &pb.Path_Element{
			Type: &i.kind,
		}
		// At most one of {Name,Id} should be set.
		// Neither will be set for incomplete keys.
		if i.stringID != "" {
			e[n].Name = &i.stringID
		} else if i.intID != 0 {
			e[n].Id = &i.intID
		}
	}
	return &pb.Reference{
		App:       proto.String(k.appID),
		NameSpace: namespace,
		Path:      &pb.Path{
			Element: e,
		},
	}
}

// referenceValueToKey is the same as protoToKey except the input is a
// PropertyValue_ReferenceValue instead of a Reference.
func referenceValueToKey(r *pb.PropertyValue_ReferenceValue) (k *Key, err os.Error) {
	appID := proto.GetString(r.App)
	namespace := proto.GetString(r.NameSpace)
	for _, e := range r.Pathelement {
		k = &Key{
			appID:     appID,
			namespace: namespace,
			parent:    k,
			kind:      proto.GetString(e.Type),
			stringID:  proto.GetString(e.Name),
			intID:     proto.GetInt64(e.Id),
		}
		if !k.valid() {
			return nil, ErrInvalidKey
		}
	}
	return
}
// keyToReferenceValue is the same as toProto except the output is a
// PropertyValue_ReferenceValue instead of a Reference.
func keyToReferenceValue(k *Key) *pb.PropertyValue_ReferenceValue {
	ref := keyToProto(k)
	pe := make([]*pb.PropertyValue_ReferenceValue_PathElement, len(ref.Path.Element))
	for i, e := range ref.Path.Element {
		pe[i] = &pb.PropertyValue_ReferenceValue_PathElement{
			Type: e.Type,
			Id:   e.Id,
			Name: e.Name,
		}
	}
	return &pb.PropertyValue_ReferenceValue{
		App:         ref.App,
		NameSpace:   ref.NameSpace,
		Pathelement: pe,
	}
}

// multiKeyToProto is a batch version of keyToProto.
func multiKeyToProto(appID string, key []*Key) []*pb.Reference {
	ret := make([]*pb.Reference, len(key))
	for i, k := range key {
		ret[i] = keyToProto(k)
	}
	return ret
}

// multiValid is a batch version of Key.valid. It returns an os.Error, not a
// []bool.
func multiValid(key []*Key) os.Error {
	invalid := false
	for _, k := range key {
		if !k.valid() {
			invalid = true
			break
		}
	}
	if !invalid {
		return nil
	}
	err := make(ErrMulti, len(key))
	for i, k := range key {
		if !k.valid() {
			err[i] = ErrInvalidKey
		}
	}
	return err
}

/*
// TODO: Validate the namespace or let the backend return an error?

var namespaceRegexp = regexp.MustCompile(`^[0-9A-Za-z._\-]{0,100}$`)

// validNamespace validates a namespace using the same rule used by
// the Python runtime: it must match the pattern `^[0-9A-Za-z._\-]{0,100}$`.
func validNamespace(namespace string) bool {
	return namespaceRegexp.Match([]byte(namespace))
}
*/