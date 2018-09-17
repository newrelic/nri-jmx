package main

import (
	"testing"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
)

func Test_checkMetricList(t *testing.T) {
	// Ensure we hit the
	args.MetricLimit = 2

	i, err := integration.New("kafka", "1.0.0")
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		t.FailNow()
	}

	e1, err := i.Entity("test_1", "domain")
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		t.FailNow()
	}

	set1 := e1.NewMetricSet("testSample")
	set1.SetMetric("new_metric", 4, metric.GAUGE)

	e2, err := i.Entity("test_2", "domain")
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		t.FailNow()
	}

	set2 := e2.NewMetricSet("testSample")
	set2.SetMetric("new_metric", 4, metric.GAUGE)
	set2.SetMetric("other_metric", 5, metric.GAUGE)

	out := checkMetricLimit(i.Entities)

	if length := len(out); length != 1 {
		t.Errorf("Expected 1 entity got %d", length)
		t.FailNow()
	}

	if out[0] != e1 {
		t.Errorf("Expected entity '%+v' got '%+v'", e1, out[0])
	}
}
