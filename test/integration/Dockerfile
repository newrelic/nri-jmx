FROM golang:1.16 as builder
ARG CGO_ENABLED=0
ARG NRJMX_VERSION=1.5.3
WORKDIR /go/src/github.com/newrelic/nri-jmx
COPY . .
RUN make clean compile
WORKDIR /
ADD http://download.newrelic.com/infrastructure_agent/binaries/linux/noarch/nrjmx_linux_${NRJMX_VERSION}_noarch.tar.gz ./
RUN tar -xzf nrjmx_linux_${NRJMX_VERSION}_noarch.tar.gz

FROM alpine:latest
RUN apk --update add openjdk8-jre
COPY --from=builder /usr/bin/nrjmx /usr/bin/nrjmx
COPY --from=builder /usr/bin/nrjmx.jar /usr/bin/nrjmx.jar
COPY --from=builder /usr/bin/jmxterm.jar /usr/bin/jmxterm.jar
COPY --from=builder /go/src/github.com/newrelic/nri-jmx/bin /
COPY test/integration/jmx-conf/ /
CMD ["sleep", "1h"]
