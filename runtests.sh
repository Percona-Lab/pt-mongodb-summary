#!/bin/bash
export BASEDIR=$(git rev-parse --show-toplevel)
if [ ! -d "${GOPATH}/src/github.com/golang/mock/gomock" ] 
then
    echo "Installing gomock & mockgen"
    go get github.com/golang/mock/gomock
    go get github.com/golang/mock/mockgen
fi
cd $BASEDIR

withmock go test -v 

#TODO withmock is producing a weird behavior in os.Getwd and coverage. 
#Still don't know how to fix it
#withmock go test -v -coverprofile=${BASEDIR}/coverage.out
#go tool cover -func=coverage.out

