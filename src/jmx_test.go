package main

import (
	"testing"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
)

func Test_checkMetricList_NoPanic(t *testing.T) {
	// Ensure we hit the
	args.MetricLimit = 1

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Unexpected panic: %s", r)
		}
	}()

	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		t.FailNow()
	}

	e, err := i.Entity("test", "domain")
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		t.FailNow()
	}

	set := e.NewMetricSet("testSample")
	set.SetMetric("new_metric", 4, metric.GAUGE)

	checkMetricLimit(i.Entities)
}
