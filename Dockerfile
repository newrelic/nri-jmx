FROM maven:3-jdk-11 as builder-mvn
RUN git clone https://github.com/newrelic/nrjmx.git && \
    cd nrjmx && \
    mvn package -DskipTests -P \!deb,\!rpm,\!test,\!tarball

FROM golang:1.25.0 as builder
COPY . /go/src/github.com/newrelic/nri-jmx/
RUN cd /go/src/github.com/newrelic/nri-jmx && \
    make && \
    strip ./bin/nri-jmx

FROM newrelic/infrastructure:latest
ENV NRIA_IS_FORWARD_ONLY true
ENV NRIA_K8S_INTEGRATION true
RUN apk --update add openjdk8-jre
COPY --from=builder-mvn /nrjmx/bin/nrjmx /usr/bin/nrjmx
COPY --from=builder-mvn /nrjmx/bin/nrjmx.jar /usr/bin/nrjmx.jar
COPY --from=builder /go/src/github.com/newrelic/nri-jmx/bin/nri-jmx /nri-sidecar/newrelic-infra/newrelic-integrations/bin/nri-jmx
USER 1000
