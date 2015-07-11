# `Good news, everyone! We moved to GitHub:` https://github.com/gorilla #


---

[Gorilla](http://www.gorillatoolkit.org/) is a web toolkit for the [Go programming language](http://golang.org/).
Currently these packages are available:

  * [gorilla/context](http://www.gorillatoolkit.org/pkg/context) stores global request variables.
  * [gorilla/mux](http://www.gorillatoolkit.org/pkg/mux) is a powerful URL router and dispatcher.
  * [gorilla/reverse](http://www.gorillatoolkit.org/pkg/reverse) produces reversible regular expressions for regexp-based muxes.
  * [gorilla/rpc](http://www.gorillatoolkit.org/pkg/rpc) implements RPC over HTTP with codec for [JSON-RPC](http://www.gorillatoolkit.org/pkg/rpc/json).
  * [gorilla/schema](http://www.gorillatoolkit.org/pkg/schema) converts form values to a struct.
  * [gorilla/securecookie](http://www.gorillatoolkit.org/pkg/securecookie) encodes and decodes authenticated and optionally encrypted cookie values.
  * [gorilla/sessions](http://www.gorillatoolkit.org/pkg/sessions) saves cookie and filesystem sessions and allows custom session backends.

And maybe [a few others](http://www.gorillatoolkit.org/pkg/).


---

To install, run "go get" pointing to a package. For example:
```
$ go get github.com/gorilla/mux
```
Or clone a repository and use the source code directly:
```
$ git clone git://github.com/gorilla/mux.git
```