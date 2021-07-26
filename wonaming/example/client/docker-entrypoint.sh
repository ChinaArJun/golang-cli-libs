#!/bin/bash

GO_WORK_DIR=${GO_WORK_DIR:-$GOPATH/src}
cd ${GO_WORK_DIR}
exec /tmp/exe $ARGUMENTS

