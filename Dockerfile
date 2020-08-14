ARG GO_VERSION=1.13.1
FROM golang:${GO_VERSION} AS builder
LABEL maintainer="ericchou19831101@msn.com"


ARG version="local"
ARG application="sensu_exporter"

ENV GOOS=linux \
    GO111MODULE="on" \
    CGO_ENABLED=0

WORKDIR /src
COPY . ./

RUN go build -ldflags "-w -s -X main.version=${version}" -o ${application}

RUN curl -H "X-JFrog-Art-Api:mySecretToken" \
        --progress-bar --upload-file ${application} \
         "https://artifactory.mycompany.com/artifactory/repo/${application}/${version}/${application}"
