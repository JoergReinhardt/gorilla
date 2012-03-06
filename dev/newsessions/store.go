// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sessions

import (
	"io"
	"net/http"
	"os"
	"sync"

	"code.google.com/p/gorilla/securecookie"
)

// Store is an interface for custom session stores.
type Store interface {
	Get(r *http.Request, name string) (*Session, error)
	New(r *http.Request, name string) (*Session, error)
	Save(r *http.Request, w http.ResponseWriter, s *Session) error
}

// CookieStore ----------------------------------------------------------------

// NewCookieStore returns a new CookieStore.
//
// Keys are defined in pairs to allow key rotation, but the common case is
// to set a single authentication key and optionally an encryption key.
//
// The first key in a pair is used for authentication and the second for
// encryption. The encryption key can be set to nil or omitted in the last
// pair, but the authentication key is required in all pairs.
//
// It is recommended to use an authentication key with 32 or 64 bytes.
// The encryption key, if set, must be either 16, 24, or 32 bytes to select
// AES-128, AES-192, or AES-256 modes.
//
// Use the convenience function securecookie.GenerateRandomKey() to create
// strong keys.
func NewCookieStore(keyPairs ...[]byte) *CookieStore {
	return &CookieStore{
		Codecs:  securecookie.CodecsFromPairs(keyPairs...),
		Options: &Options{
			Path:   "/",
			MaxAge: 86400 * 30,
		},
	}
}

// CookieStore stores sessions using secure cookies.
type CookieStore struct {
	Codecs  []securecookie.Codec
	Options *Options // default configuration
}

// Get returns a session for the given name after adding it to the registry.
//
// It returns a new session if the sessions doesn't exist. Access IsNew on
// the session to check if it is an existing session or a new one.
//
// It returns a new session and an error if the session exists but could
// not be decoded.
func (s *CookieStore) Get(r *http.Request, name string) (*Session, error) {
	return GetRegistry(r).Get(s, name)
}

// New returns a session for the given name without adding it to the registry.
//
// The difference between New() and Get() is that calling New() twice will
// decode the session data twice, while Get() registers and reuses the same
// decoded session after the first call.
func (s *CookieStore) New(r *http.Request, name string) (*Session, error) {
	session := NewSession(s, name)
	session.IsNew = true
	var err error
	if c, errCookie := r.Cookie(name); errCookie == nil {
		err = securecookie.DecodeMulti(name, c.Value, &session.Values,
			s.Codecs...)
		if err == nil {
			session.IsNew = false
		}
	}
	return session, err
}

// Save adds a single session to the response.
func (s *CookieStore) Save(r *http.Request, w http.ResponseWriter,
	session *Session) error {
	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values,
		s.Codecs...)
	if err != nil {
		return err
	}
	options := s.Options
	if session.Options != nil {
		options = session.Options
	}
	cookie := &http.Cookie{
		Name:     session.Name(),
		Value:    encoded,
		Path:     options.Path,
		Domain:   options.Domain,
		MaxAge:   options.MaxAge,
		Secure:   options.Secure,
		HttpOnly: options.HttpOnly,
	}
	http.SetCookie(w, cookie)
	return nil
}

// FilesystemStore ------------------------------------------------------------

var fileMutex sync.RWMutex

// NewCookieStore returns a new CookieStore.
//
// The path argument is the directory where sessions will be saved. If empty
// it will use os.TempDir().
//
// See NewCookieStore() for a description of the other parameters.
func NewFilesystemStore(path string, keyPairs ...[]byte) *FilesystemStore {
	if path == "" {
		path = os.TempDir()
	}
	if path[len(path)-1] != '/' {
		path += "/"
	}
	return &FilesystemStore{
		path:    path,
		Codecs:  securecookie.CodecsFromPairs(keyPairs...),
		Options: &Options{
			Path:   "/",
			MaxAge: 86400 * 30,
		},
	}
}

// FilesystemStore stores sessions in the filesystem.
type FilesystemStore struct {
	path    string
	Codecs  []securecookie.Codec
	Options *Options // default configuration
}

// Get returns a session for the given name after adding it to the registry.
//
// See CookieStore.Get().
func (s *FilesystemStore) Get(r *http.Request, name string) (*Session, error) {
	return GetRegistry(r).Get(s, name)
}

// New returns a session for the given name without adding it to the registry.
//
// See CookieStore.New().
func (s *FilesystemStore) New(r *http.Request, name string) (*Session, error) {
	session := NewSession(s, name)
	session.IsNew = true
	var err error
	if c, errCookie := r.Cookie(name); errCookie == nil {
		err = securecookie.DecodeMulti(name, c.Value, &session.ID, s.Codecs...)
		if err == nil {
			err = s.readFile(session)
			if err == nil {
				session.IsNew = false
			}
		}
	}
	return session, err
}

// Save adds a single session to the response.
func (s *FilesystemStore) Save(r *http.Request, w http.ResponseWriter,
	session *Session) error {
	if session.ID == nil {
		session.ID = securecookie.GenerateRandomKey(32)
	}
	if err := s.writeFile(session); err != nil {
		return err
	}
	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID,
		s.Codecs...)
	if err != nil {
		return err
	}
	options := s.Options
	if session.Options != nil {
		options = session.Options
	}
	cookie := &http.Cookie{
		Name:     session.Name(),
		Value:    encoded,
		Path:     options.Path,
		Domain:   options.Domain,
		MaxAge:   options.MaxAge,
		Secure:   options.Secure,
		HttpOnly: options.HttpOnly,
	}
	http.SetCookie(w, cookie)
	return nil
}

// writeFile writes encoded session.Values in a file.
func (s *FilesystemStore) writeFile(session *Session) error {
	if len(session.Values) == 0 {
		// Don't need to write anything.
		return nil
	}
	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values,
		s.Codecs...)
	if err != nil {
		return err
	}
	filename := s.path + "session_" + string(session.ID)
	fileMutex.Lock()
	defer fileMutex.Unlock()
	fp, err2 := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0600)
	if err2 != nil {
		return err2
	}
	if _, err = fp.Write([]byte(encoded)); err != nil {
		return err
	}
	fp.Close()
	return nil
}

// readFile reads a file and decodes its content into session.Values.
func (s *FilesystemStore) readFile(session *Session) error {
	filename := s.path + "session_" + string(session.ID)
	fp, err := os.OpenFile(filename, os.O_RDONLY, 0400)
	if err != nil {
		return err
	}
	defer fp.Close()
	var fdata []byte
	buf := make([]byte, 128)
	for {
		var n int
		n, err = fp.Read(buf[0:])
		fdata = append(fdata, buf[0:n]...)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	if err = securecookie.DecodeMulti(session.Name(), string(fdata),
		&session.Values, s.Codecs...); err != nil {
		return err
	}
	return nil
}
