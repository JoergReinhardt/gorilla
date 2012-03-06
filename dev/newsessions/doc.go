// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package gorilla/sessions provides cookie sessions and infrastructure for
custom session backends.

The key features are:

	* Simple API: use it as an easy way to set signed (and optionally
	  encrypted) cookies.
	* Built-in backends to store sessions in cookies or the filesystem.
	* Multiple sessions per request, even using different backends.
	* Convenient way to switch session persistency (aka "remember me") and set
	  other attributes.
	* Flash messages: session values that last until read.
	* Mechanism to rotate authentication and encryption keys.
	* Interfaces and infrastructure for custom session backends: sessions
	  from different backends are registered in the same place and retrieved
	  and saved using a common API.

Let's start with an example that shows the sessions API in a nutshell:

	import (
		"net/http"
		"code.google.com/p/gorilla/sessions"
	)

	var store = sessions.NewCookieStore([]byte("something-very-secret"))

	func MyHandler(w http.ResponseWriter, r *http.Request) {
		// Get a session. We're ignoring the error resulted from decoding an
		// existing session: Get() always returns a session, even if empty.
		session, _ := store.Get(r, "session-name")
		// Set some session values.
		session.Values["foo"] = "bar"
		session.Values[42] = 43
		// Save it.
		session.Save(r, w)
	}

First we initialize a session store calling NewCookieStore() and passing a
secret key used to authenticate the session. Inside the handler, we call
store.Get() to retrieve an existing session or a new one. Then we set some
session values in session.Values, which is a map[interface{}]interface{}.
And finally we call session.Save() to save the session in the response.

That's all you need to know for the basic usage. Let's take a look at other
options, starting with flash messages.

Flash messages are session values that last until read. The term appeared with
Ruby On Rails a few years back. When we request a flash message, it is removed
from the session. To add a flash, call session.AddFlash(), and to get all
flashes, call session.Flashes(). Here is an example:

	func MyHandler(w http.ResponseWriter, r *http.Request) {
		// Get a session.
		session, _ := store.Get(r, "session-name")
		// Get the previously flashes, if any.
		if flashes := session.Flashes(); len(flashes) > 0 {
			// Just print the flash values.
			fmt.Fprint(w, "%v", flashes)
		} else {
			// Set a new flash.
			session.AddFlash("Hello, flash messages world!")
			fmt.Fprint(w, "No flashes found.")
		}
		session.Save(r, w)
	}

Flash messages are useful to set information to be read after a redirection,
like after form submissions.

By default, session cookies last for a month. This is probably too long for
some cases, but it is easy to change this and other attributes during
runtime. Sessions can be configured individually or the store can be
configured and then all sessions saved using it will use that configuration.
We access session.Options or store.Options to set a new configuration. The
fields are basically a subset of http.Cookie fields. Let's change the
maximum age of a session to one week:

	session.Options = &sessions.Options{
		Path:   "/",
		MaxAge: 86400 * 7,
	}

Sometimes we may want to change authentication and/or encryption keys without
breaking existing sessions. The CookieStore supports key rotation, and to use
it you just need to set multiple authentication and encryption keys, in pairs,
to be tested in order:

	var store = sessions.NewCookieStore(
		[]byte("new-authentication-key"),
		[]byte("new-encryption-key"),
		[]byte("old-authentication-key"),
		[]byte("old-encryption-key"),
	)

New sessions will be saved using the first pair. Old sessions can still be
read because the first pair will fail, and the second will be tested. This
makes it easy to "rotate" secret keys and still be able to validate existing
sessions. Note: for all pairs the encryption key is optional; set it to nil
or omit it and and encryption won't be used.

Multiple sessions can be used in the same request, even with different
session backends. When this happens, calling Save() on each session
individually would be cumbersome, so we have a way to save all sessions
at once: it's sessions.Save(). Here's an example:

	var store = sessions.NewCookieStore([]byte("something-very-secret"))

	func MyHandler(w http.ResponseWriter, r *http.Request) {
		// Get a session and set a value.
		session1, _ := store.Get(r, "session-one")
		session1.Values["foo"] = "bar"
		// Get another session and set another value.
		session2, _ := store.Get(r, "session-two")
		session2.Values[42] = 43
		// Save all sessions.
		sessions.Save(r, w)
	}

This is possible because when we call Get() from a session store, it adds the
session to a common registry. Save() uses it to save all registered sessions.
*/
package sessions
