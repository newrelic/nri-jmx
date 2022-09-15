/*
 * Copyright 2022 New Relic Corporation. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

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
		input        map[string]interface{}
		output       *attributeRequest
		expectedFail bool
	}{
		{
			map[string]interface{}{"attr": "testattr", "metric_type": "gauge", "metric_name": "testmetricname"},
			&attributeRequest{attrRegexp: regexp.MustCompile("attr=testattr$"), metricName: "testmetricname", metricType: metric.GAUGE},
			false,
		},
		{
			map[string]interface{}{"attr": "testattr", "metric_name": "testmetricname"},
			&attributeRequest{attrRegexp: regexp.MustCompile("attr=testattr$"), metricName: "testmetricname", metricType: -1},
			false,
		},
		{
			map[string]interface{}{"attr": "testattr"},
			&attributeRequest{attrRegexp: regexp.MustCompile("attr=testattr$"), metricType: -1},
			false,
		},
		{
			map[string]interface{}{"attr": "testattr", "attr_regex": "testattrregex"},
			&attributeRequest{attrRegexp: regexp.MustCompile("attr=testattr$"), metricType: -1},
			true,
		},
		{
			map[string]interface{}{},
			&attributeRequest{attrRegexp: regexp.MustCompile("attr=testattr$"), metricType: -1},
			true,
		},
		{
			map[string]interface{}{"attr_regex": "testattr"},
			&attributeRequest{attrRegexp: regexp.MustCompile("attr=testattr$"), metricType: -1},
			false,
		},
		{
			map[string]interface{}{"attr": "testattr"},
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
					map[string]interface{}{
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
					map[string]interface{}{
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

func TestParseCollectionDefinitionJSON(t *testing.T) {
	configJSON := `
          {
              "collect": [
                  {
                      "domain": "com.demo.app",
                      "event_type": "JMXAnnotationSample",
                      "beans": [
                          {
                              "query": "name=SystemStatusExample",
                              "exclude_regex": [
                                "Random.*"
                              ],
                              "attributes": [
                                  {
                                    "attr_regex": "Random.*",
                                    "metric_name": "t.test",
                                    "metric_type": "rate"
                                  }
                              ]
                          }
                      ]
                  }
              ]
          }`

	expectedDomains := []*domainDefinition{
		{
			domain:    "com.demo.app",
			eventType: "JMXAnnotationSample",
			beans: []*beanRequest{
				{
					beanQuery: "name=SystemStatusExample",
					exclude: []*regexp.Regexp{
						regexp.MustCompile("Random.*"),
					},
					attributes: []*attributeRequest{
						{
							attrRegexp: regexp.MustCompile(`attr=Random.*$`),
							metricName: "t.test",
							metricType: metric.RATE,
						},
					},
				},
			},
		},
	}

	c, err := parseJSON(configJSON)
	assert.NoError(t, err)

	actualDomains, err := parseCollectionDefinition(c)
	assert.NoError(t, err)
	assert.Equal(t, expectedDomains, actualDomains)
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
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		t.FailNow()
	}
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
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		t.FailNow()
	}
	_, err = parseCollectionDefinition(c)
	if err == nil {
		t.Error("Expected error")
	}
}
