// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package main

var usage = `%s runs a CRISP-1.0 server.

It takes 1 argument, an address on which to serve.  Addresses
may either be in the form

	@path/to/somewhere

or

	host:port

The first form specifies a unix domain socket by a prefix '@'.

%s takes the following flags.

`
