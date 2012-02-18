// Copyright 2011 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package datastore

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"strconv"
	"strings"

	"appengine"
	"code.google.com/p/goprotobuf/proto"

	pb "appengine_internal/datastore"
)

// Key represents the datastore key for a stored entity, and is immutable.
type Key struct {
	kind      string
	stringID  string
	intID     int64
	parent    *Key
	appID     string
	namespace string
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

// Parent returns the key's parent key, which may be nil.
func (k *Key) Parent() *Key {
	return k.parent
}

// AppID returns the key's application ID.
func (k *Key) AppID() string {
	return k.appID
}

// Namespace returns the key's namespace.
func (k *Key) Namespace() string {
	return k.namespace
}

// Incomplete returns whether the key does not refer to a stored entity.
// In particular, whether the key has a zero StringID and a zero IntID.
func (k *Key) Incomplete() bool {
	return k.stringID == "" && k.intID == 0
}

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

// Equal returns whether two keys are equal.
func (k *Key) Equal(o *Key) bool {
	for k != nil && o != nil {
		if k.kind != o.kind || k.stringID != o.stringID || k.intID != o.intID || k.appID != o.appID || k.namespace != o.namespace {
			return false
		}
		k, o = k.parent, o.parent
	}
	return k == o
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
	} else {
		b.WriteString(k.namespace)
	}
	b.WriteByte('/')
	b.WriteString(k.kind)
	b.WriteByte(',')
	if k.stringID != "" {
		b.WriteString(k.stringID)
	} else {
		b.WriteString(strconv.FormatInt(k.intID, 10))
	}
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

func (k *Key) GobEncode() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(keyToGobKey(k)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (k *Key) GobDecode(buf []byte) error {
	gk := new(gobKey)
	if err := gob.NewDecoder(bytes.NewBuffer(buf)).Decode(gk); err != nil {
		return err
	}
	*k = *gobKeyToKey(gk)
	return nil
}

func (k *Key) MarshalJSON() ([]byte, error) {
	return []byte(`"` + k.Encode() + `"`), nil
}

func (k *Key) UnmarshalJSON(buf []byte) error {
	if len(buf) < 2 || buf[0] != '"' || buf[len(buf)-1] != '"' {
		return errors.New("datastore: bad JSON key")
	}
	k2, err := DecodeKey(string(buf[1 : len(buf)-1]))
	if err != nil {
		return err
	}
	*k = *k2
	return nil
}

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

// DecodeKey decodes a key from the opaque representation returned by Encode.
func DecodeKey(encoded string) (*Key, error) {
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
	return &Key{
		kind:     kind,
		stringID: stringID,
		intID:    intID,
		parent:   parent,
		appID:    c.FullyQualifiedAppID(),
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
	// A namespace must match regexp.MustCompile(`^[0-9A-Za-z._\-]{0,100}$`)
	for p := parent; p != nil && namespace == ""; p = p.parent {
		namespace = p.namespace
	}
	k := NewKey(c, kind, stringID, intID, parent)
	k.namespace = namespace
	return k
}
