package main

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/kr/pretty"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
)

func TestRunCollection(t *testing.T) {

	jmxQueryFunc = func(name string, timeout int) (map[string]interface{}, error) {
		outmap := map[string]interface{}{
			"java.lang:test1=test1,test2=test2,attr=testattr": "testresult",
		}

		return outmap, nil
	}

	collection := []*domainDefinition{
		{
			domain:    "java.lang",
			eventType: "TestEvent",
			beans: []*beanRequest{
				{
					beanQuery: "test1=test1,test2=test2",
					attributes: []*attributeRequest{
						{
							attrRegexp: regexp.MustCompile("attr=testattr$"),
							metricName: "testName",
							metricType: metric.ATTRIBUTE,
						},
					},
				},
			},
		},
	}

	expectedMetrics := map[string]interface{}{
		"event_type":  "TestEvent",
		"entityName":  "domain:java.lang",
		"displayName": "java.lang",
		"host":        "",
		"testName":    "testresult",
		"query":       "test1=test1,test2=test2",
		"bean":        "test1=test1,test2=test2",
		"key:test1":   "test1",
		"key:test2":   "test2",
	}

	i, _ := integration.New("jmxtest", "0.1.0")

	runCollection(collection, i)

	if !reflect.DeepEqual(expectedMetrics, i.Entities[0].Metrics[0].Metrics) {
		fmt.Println(pretty.Diff(expectedMetrics, i.Entities[0].Metrics[0].Metrics))
		t.Error("Failed to produce expected metrics")
	}

}
func TestGenerateEventType(t *testing.T) {
	i, _ := integration.New("jmx", "0.1.0")
	logger = i.Logger()
	testCases := []struct {
		input       string
		expectedOut string
		expectedErr bool
	}{
		{"java.lang", "JavaLangSample", false},
		{"java", "JavaSample", false},
		{"java.lang.test", "JavaLangTestSample", false},
		{"java.*", "", true},
	}

	for _, tc := range testCases {
		out, err := generateEventType(tc.input)
		if (err != nil) != tc.expectedErr {
			t.Errorf("Bad error case for %s", tc.input)
		}
		if out != tc.expectedOut {
			t.Errorf("Expected event type %s, got %s", tc.expectedOut, out)
		}
	}
}

func TestGetBeanName(t *testing.T) {
	testCases := []struct {
		input       string
		expectedOut string
	}{
		{"type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=Min", "type=RequestMetrics,name=TotalTimeMs,request=Fetch"},
		{"type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=Min.2", "type=RequestMetrics,name=TotalTimeMs,request=Fetch"},
		{"type=Request,attr=Test", "type=Request"},
	}

	for _, tc := range testCases {
		out, err := getBeanName(tc.input)
		if err != nil {
			t.Error(err)
		}
		if out != tc.expectedOut {
			t.Errorf("Expected metric name %s, got %s", tc.expectedOut, out)
		}
	}
}

func TestGetAttrName(t *testing.T) {
	testCases := []struct {
		input       string
		expectedOut string
	}{
		{"type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=Min", "Min"},
		{"type=RequestMetrics,name=TotalTimeMs,request=Fetch,attr=Min.2", "Min.2"},
		{"type=Request,attr=Test", "Test"},
	}

	for _, tc := range testCases {
		out, err := getAttrName(tc.input)
		if err != nil {
			t.Error(err)
		}
		if out != tc.expectedOut {
			t.Errorf("Expected metric name %s, got %s", tc.expectedOut, out)
		}
	}
}

func TestSplitBeanName(t *testing.T) {
	testCases := []struct {
		input        string
		expectedOut1 string
		expectedOut2 string
	}{
		{"java.lang:test=test", "java.lang", "test=test"},
		{"java.lang:test=test,test2=test2", "java.lang", "test=test,test2=test2"},
	}

	for _, tc := range testCases {
		out1, out2, err := splitBeanName(tc.input)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		if out1 != tc.expectedOut1 && out2 != tc.expectedOut2 {
			t.Errorf("Expected %s and %s, got %s and %s", out1, out2, tc.expectedOut1, tc.expectedOut2)
		}
	}
}

func TestInferMetricType(t *testing.T) {
	testCases := []struct {
		input       interface{}
		expectedOut metric.SourceType
	}{
		{1, metric.GAUGE},
		{1.0, metric.GAUGE},
		{float32(1.0), metric.GAUGE},
		{"true", metric.ATTRIBUTE},
		{true, metric.ATTRIBUTE},
	}

	for _, tc := range testCases {
		if inferMetricType(tc.input) != tc.expectedOut {
			t.Errorf("Expected metric type %d, got %d", tc.expectedOut, inferMetricType(tc.input))
		}
	}
}

func TestInsertMetric(t *testing.T) {
	i, _ := integration.New("jmx", "0.1.0")
	e, _ := i.Entity("testEntity", "test")
	m := e.NewMetricSet("testSet")
	key := "test1=test1,test2=test2,attr=testattr"
	val := 1
	ar := &attributeRequest{
		attrRegexp: regexp.MustCompile("attr=testattr$"),
		metricName: "Test.Metric.Name",
		metricType: metric.GAUGE,
	}

	expectedMetrics := map[string]interface{}{
		"event_type":       "testSet",
		"Test.Metric.Name": 1.0,
	}

	insertMetric(key, val, ar, m)

	if !reflect.DeepEqual(m.Metrics, expectedMetrics) {
		fmt.Println(pretty.Diff(m.Metrics, expectedMetrics))
		t.Errorf("Did not get expected metric set")
	}
}

func TestInsertDomainMetrics(t *testing.T) {
	i, _ := integration.New("jmx", "0.1.0")
	args = argumentList{}
	args.JmxHost = "localhost"
	domain := "java.lang"
	beanAttrVals := []*beanAttrValuePair{
		{
			beanAttr: "test1=test1,test2=test2,attr=testattr",
			value:    1.0,
		},
		{
			beanAttr: "test1=test1,test2=test2,attr=testattr2",
			value:    "test",
		},
	}
	request := &beanRequest{
		beanQuery: "test1=test1,test2=test2",
		exclude: []*regexp.Regexp{
			regexp.MustCompile("wontmatch"),
		},
		attributes: []*attributeRequest{
			{
				attrRegexp: regexp.MustCompile("attr=testattr2$"),
				metricName: "testmetric2",
				metricType: metric.ATTRIBUTE,
			},
		},
	}

	eventType := "TestEventTypeSample"

	insertDomainMetrics(eventType, domain, beanAttrVals, request, i)

	expectedMetrics := map[string]interface{}{
		"event_type":  "TestEventTypeSample",
		"entityName":  "domain:java.lang",
		"displayName": "java.lang",
		"host":        "localhost",
		"query":       "test1=test1,test2=test2",
		"testmetric2": "test",
		"bean":        "test1=test1,test2=test2",
		"key:test1":   "test1",
		"key:test2":   "test2",
	}

	if !reflect.DeepEqual(i.Entities[0].Metrics[0].Metrics, expectedMetrics) {
		fmt.Println(pretty.Diff(i.Entities[0].Metrics[0].Metrics, expectedMetrics))
		t.Error("Expected different metrics")
	}
}

func TestHandleResponse(t *testing.T) {
	eventType := "TestSample"
	request := &beanRequest{
		beanQuery:  "test1=test1,test2=test2",
		exclude:    []*regexp.Regexp{},
		attributes: []*attributeRequest{},
	}
	response := map[string]interface{}{
		"test.domain:test1=test1,test2=test2,attr=test3": "test4",
	}
	i, _ := integration.New("jmx", "0.1.0")

	handleResponse(eventType, request, response, i)

}
