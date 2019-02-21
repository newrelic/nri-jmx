FROM golang:1.10 as builder
RUN go get -d github.com/newrelic/nri-jmx/... && \
    cd /go/src/github.com/newrelic/nri-jmx && \
    make && \
    strip ./bin/nr-jmx

FROM newrelic/infrastructure:latest
COPY --from=builder /go/src/github.com/newrelic/nri-jmx/bin/nr-jmx /var/db/newrelic-infra/newrelic-integrations/bin/nr-jmx
COPY --from=builder /go/src/github.com/newrelic/nri-jmx/jmx-definition.yml /var/db/newrelic-infra/newrelic-integrations/definition.yaml