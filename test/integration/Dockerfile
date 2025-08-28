FROM golang:1.25.0 as builder
ARG CGO_ENABLED=0
ARG NRJMX_VERSION
WORKDIR /go/src/github.com/newrelic/nri-jmx
COPY . .
RUN make clean compile
WORKDIR /

ADD https://github.com/newrelic/nrjmx/releases/download/v${NRJMX_VERSION}/nrjmx_linux_${NRJMX_VERSION}_noarch.tar.gz ./
RUN tar -xzf nrjmx_linux_${NRJMX_VERSION}_noarch.tar.gz

FROM alpine:latest
RUN apk --update add openjdk8-jre
COPY --from=builder /usr/bin/nrjmx /usr/bin/nrjmx
COPY --from=builder /usr/lib/nrjmx/nrjmx.jar /usr/lib/nrjmx/nrjmx.jar
COPY --from=builder /go/src/github.com/newrelic/nri-jmx/bin /
COPY test/integration/jmx-conf/ /
CMD ["sleep", "1h"]
