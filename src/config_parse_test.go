package main

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/kr/pretty"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
)

func TestParseYaml(t *testing.T) {
	testCases := []struct {
		file         string
		expectedFail bool
	}{
		{"../test/data/test-sample.yml", false},
		{"../test/data/test-sample-bad.yml", true},
		{"../test/data/test-sample-nonexistant.yml", true},
	}

	for _, tc := range testCases {
		_, err := parseYaml(tc.file)
		if (err != nil) != tc.expectedFail {
			t.Error("Did not get expected error state")
		}
	}
}

func TestParseAttributeFromString(t *testing.T) {
	testCases := []struct {
		input         string
		output        *attributeRequest
		expectedError bool
	}{
		{"Testattribute", &attributeRequest{attrRegexp: regexp.MustCompile("attr=Testattribute$")}, false},
		{`weird.string([)]`, &attributeRequest{attrRegexp: regexp.MustCompile(`attr=weird\.string\(\[\)\]$`)}, false},
	}

	for _, tc := range testCases {
		rq, err := parseAttributeFromString(tc.input)
		if !reflect.DeepEqual(rq, tc.output) && (err != nil) != tc.expectedError {
			t.Errorf("Failed to create attribute from string %s", tc.input)
		}
	}
}

func TestParseAttributeFromMap(t *testing.T) {
	testCases := []struct {
		input        map[interface{}]interface{}
		output       *attributeRequest
		expectedFail bool
	}{
		{
			map[interface{}]interface{}{"attr": "testattr", "metric_type": "gauge", "metric_name": "testmetricname"},
			&attributeRequest{attrRegexp: regexp.MustCompile("attr=testattr$"), metricName: "testmetricname", metricType: metric.GAUGE},
			false,
		},
		{
			map[interface{}]interface{}{"attr": "testattr", "metric_name": "testmetricname"},
			&attributeRequest{attrRegexp: regexp.MustCompile("attr=testattr$"), metricName: "testmetricname", metricType: -1},
			false,
		},
		{
			map[interface{}]interface{}{"attr": "testattr"},
			&attributeRequest{attrRegexp: regexp.MustCompile("attr=testattr$"), metricType: -1},
			false,
		},
		{
			map[interface{}]interface{}{"attr": "testattr", "attr_regex": "testattrregex"},
			&attributeRequest{attrRegexp: regexp.MustCompile("attr=testattr$"), metricType: -1},
			true,
		},
		{
			map[interface{}]interface{}{},
			&attributeRequest{attrRegexp: regexp.MustCompile("attr=testattr$"), metricType: -1},
			true,
		},
		{
			map[interface{}]interface{}{"attr_regex": "testattr"},
			&attributeRequest{attrRegexp: regexp.MustCompile("attr=testattr$"), metricType: -1},
			false,
		},
		{
			map[interface{}]interface{}{"attr": "testattr"},
			&attributeRequest{attrRegexp: regexp.MustCompile("attr=testattr$"), metricType: metric.DELTA},
			true,
		},
	}

	for i, tc := range testCases {
		rq, err := parseAttributeFromMap(tc.input)
		if err != nil {
			if !tc.expectedFail {
				t.Error(err)
			}
			continue
		}
		if !reflect.DeepEqual(rq, tc.output) && !tc.expectedFail {
			fmt.Println(pretty.Diff(rq, tc.output))
			t.Errorf("Not the same for test case %d", i)
		}
	}
}

func TestParseBean(t *testing.T) {
	testCases := []struct {
		input        *beanDefinitionParser
		expected     *beanRequest
		expectedFail bool
	}{
		{
			&beanDefinitionParser{
				Query: "name=test,partition=test",
			},
			&beanRequest{
				beanQuery: "name=test,partition=test",
				attributes: []*attributeRequest{
					{
						attrRegexp: regexp.MustCompile("attr=.*$"),
						metricType: -1,
					},
				},
			},
			false,
		},
		{
			&beanDefinitionParser{
				Query:   "name=test,partition=test",
				Exclude: []interface{}{"testexclude"},
			},
			&beanRequest{
				beanQuery: "name=test,partition=test",
				exclude: []*regexp.Regexp{
					regexp.MustCompile("testexclude"),
				},
				attributes: []*attributeRequest{
					{
						attrRegexp: regexp.MustCompile("attr=.*$"),
						metricType: -1,
					},
				},
			},
			false,
		},
		{
			&beanDefinitionParser{
				Query:   "name=test,partition=test",
				Exclude: []interface{}{"testexclude"},
				Attributes: []interface{}{
					map[interface{}]interface{}{
						"attr": "testattr",
					},
				},
			},
			&beanRequest{
				beanQuery: "name=test,partition=test",
				exclude: []*regexp.Regexp{
					regexp.MustCompile("testexclude"),
				},
				attributes: []*attributeRequest{
					{
						attrRegexp: regexp.MustCompile("attr=testattr$"),
						metricType: -1,
					},
				},
			},
			false,
		},
		{
			&beanDefinitionParser{
				Query:   "name=test,partition=test",
				Exclude: []interface{}{"testexclude"},
				Attributes: []interface{}{
					map[interface{}]interface{}{
						"attr":        "testattr",
						"metric_type": "gauge",
					},
				},
			},
			&beanRequest{
				beanQuery: "name=test,partition=test",
				exclude: []*regexp.Regexp{
					regexp.MustCompile("testexclude"),
				},
				attributes: []*attributeRequest{
					{
						attrRegexp: regexp.MustCompile("attr=testattr$"),
						metricType: metric.GAUGE,
					},
				},
			},
			false,
		},
	}

	for i, tc := range testCases {
		rq, err := parseBean(tc.input)
		if err != nil {
			if !tc.expectedFail {
				t.Error(err)
			}
			continue
		}

		if !reflect.DeepEqual(rq, tc.expected) && !tc.expectedFail {
			fmt.Println(pretty.Diff(rq, tc.expected))
			t.Errorf("Not the same for test case %d", i)
		}
	}
}

func TestParseCollectionDefinition(t *testing.T) {
	expectedDomains := []*domainDefinition{
		{
			domain:    "test.test",
			eventType: "TestTestSample",
			beans: []*beanRequest{
				{
					beanQuery: "test=test",
					exclude: []*regexp.Regexp{
						regexp.MustCompile("test"),
					},
					attributes: []*attributeRequest{
						{
							attrRegexp: regexp.MustCompile(`attr=test\.test$`),
							metricName: "t.test",
							metricType: metric.RATE,
						},
						{
							attrRegexp: regexp.MustCompile(`attr=test.*$`),
							metricName: "",
							metricType: -1,
						},
					},
				},
			},
		},
		{
			domain:    "test.*",
			eventType: "TestSample",
			beans: []*beanRequest{
				{
					beanQuery: "test=tester",
					exclude: []*regexp.Regexp{
						regexp.MustCompile("test2"),
					},
					attributes: []*attributeRequest{
						{
							attrRegexp: regexp.MustCompile(`attr=.*$`),
							metricName: "",
							metricType: -1,
						},
					},
				},
			},
		},
	}

	c, err := parseYaml("../test/data/test-sample.yml")
	domains, err := parseCollectionDefinition(c)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if !reflect.DeepEqual(domains, expectedDomains) {
		fmt.Println(pretty.Diff(domains, expectedDomains))
		t.Errorf("Failed to produce expected domains list.")
	}
}

func TestParseCollectionDefinition_Fail(t *testing.T) {

	c, err := parseYaml("../test/data/test-sample-bad2.yml")
	_, err = parseCollectionDefinition(c)
	if err == nil {
		t.Error("Expected error")
	}
}
