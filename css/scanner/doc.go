// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package gorilla/css/scanner scans an input and emits tokens following the CSS3
specification located at:

	http://www.w3.org/TR/css3-syntax/

To use it, create a new scanner for a given CSS string and call Next() until
the token returned has type TokenEOF or TokenError:

	s := scanner.New(myCSS)
	for {
		token := s.Next()
		if token.Type == scanner.TokenEOF || token.Type == scanner.TokenError {
			break
		}
		// Do something with the token...
	}

Note: the scanner doesn't perform lexical analysis or, in other words, it
doesn't care about the token context. It is intended to be used by a
lexer or parser.
*/
package scanner