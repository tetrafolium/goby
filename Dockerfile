FROM golang:1.14

ENV GOPATH=/go
ENV PATH=$GOPATH/bin:$PATH

ENV GO111MODULE=on

RUN mkdir -p $GOPATH/src/github.com/tetrafolium/goby
ENV GOBY_ROOT=$GOPATH/src/github.com/tetrafolium/goby

WORKDIR $GOPATH/src/github.com/tetrafolium/goby

ADD . ./

RUN go install .
