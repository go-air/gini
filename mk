#!/bin/sh

set -e
export GOPATH=`pwd`:$GOPATH
#go test gini...
go install -gcflags="-B" gini/internal/xo 
#go install gini/internal/xo 
go install gini/cmd...
