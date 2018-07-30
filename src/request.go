package main

import (
	"fmt"
	"strings"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
)

// queryResponse is a struct that contains the
// response from a JMX query
type queryResponse map[string]interface{}

// beanAttrValuePair is a convenience struct that
// contains the fully qualified bean name + attribute
// tag and also contains the value associated with
// that attribute. It facilitates passing the attribute
// and its value between functions easily
type beanAttrValuePair struct {
	beanAttr string
	value    interface{}
}

func runCollection(collection []*domainDefinition, i *integration.Integration) error {
	for _, domain := range collection {
		for _, request := range domain.beans {
			requestString := fmt.Sprintf("%s:%s", domain.domain, request.beanQuery)
			result, err := jmxQueryFunc(requestString, args.Timeout)
			if err != nil {
				logger.Errorf("Failed to retrieve metrics for request %s: %s", requestString, err)
				return err
			}
			if err := handleResponse(domain.eventType, request, result, i); err != nil {
				logger.Errorf("Failed to parse response for request %s: %s", requestString, err)
				return err
			}
		}
	}

	return nil
}

// handleResponse takes a response, filters out the excluded beans,
// sorts the responses by domain, and passes each domain off to
// insertDomainMetrics to populate the metric list
func handleResponse(eventType string, request *beanRequest, response queryResponse, i *integration.Integration) error {

	// Delete excluded mbeans
	for key := range response {
		for _, pattern := range request.exclude {
			if pattern.MatchString(key) {
				delete(response, key)
			}
		}
	}

	// If there are multiple domains, we have to create an entity for each
	// Create a map with domain as the key that returns query/value
	domainsMap := make(map[string][]*beanAttrValuePair)
	for key, val := range response {
		domain, beanAttr, err := splitBeanName(key)
		if err != nil {
			return err
		}

		domainsMap[domain] = append(domainsMap[domain], &beanAttrValuePair{beanAttr: beanAttr, value: val})
	}

	// For each domain, create an entity and a metric set
	for domain, beanAttrVals := range domainsMap {
		err := insertDomainMetrics(eventType, domain, beanAttrVals, request, i)
		if err != nil {
			return err
		}
	}

	return nil
}

// insertDomainMetrics akes a domain and a list of attr:value pairs,
// creates an entity and metric set for the domain, and populates the
// metric set for each attribute to be collected
func insertDomainMetrics(eventType string, domain string, beanAttrVals []*beanAttrValuePair, request *beanRequest, i *integration.Integration) error {

	// Create an entity for the domain
	e, err := i.Entity(domain, "domain")
	if err != nil {
		return err
	}

	// Create a metric set for the domain
	metricSet := e.NewMetricSet(eventType,
		metric.Attribute{Key: "query", Value: request.beanQuery},
		metric.Attribute{Key: "entityName", Value: "domain:" + e.Metadata.Name},
		metric.Attribute{Key: "displayName", Value: e.Metadata.Name},
		metric.Attribute{Key: "host", Value: args.JmxHost},
	)

	// For each bean/attribute returned from this domain
	for _, beanAttrVal := range beanAttrVals {
		// For each attribute we want to collect, check if it matches
		for _, attribute := range request.attributes {
			if attribute.attrRegexp.MatchString(beanAttrVal.beanAttr) {
				// If we want to collect the metric, populate the metric list
				if err := insertMetric(beanAttrVal.beanAttr, beanAttrVal.value, attribute, metricSet); err != nil {
					return err
				}
				// Once we collect this metric once, we don't want to collect it
				// as another metric that might match it
				break
			}
		}
	}

	return nil
}

// Inserts a metric into a metric set, generating metric names
// and metric types if unset
func insertMetric(key string, val interface{}, attribute *attributeRequest, metricSet *metric.Set) error {

	// Generate a metric name if unset
	var metricName string
	var err error
	if attribute.metricName == "" {
		metricName, err = generateMetricName(key)
		if err != nil {
			return err
		}
	} else {
		metricName = attribute.metricName
	}

	// Generate a metric type if unset
	var metricType metric.SourceType
	if attribute.metricType == -1 {
		metricType = inferMetricType(val)
	} else {
		metricType = attribute.metricType
	}

	// Populate the metric set with the value
	if err := metricSet.SetMetric(metricName, val, metricType); err != nil {
		return err
	}

	return nil
}

// Convenience function to split the domain:query string
// into domain and query
func splitBeanName(bean string) (string, string, error) {
	domainQuery := strings.Split(bean, ":")
	if len(domainQuery) != 2 {
		return "", "", fmt.Errorf("invalid domain:bean string %s", bean)
	}
	return domainQuery[0], domainQuery[1], nil
}

// generateEventType generates an event type from a domain string.
// The resulting event type will be used if no custom event type has been defined.
func generateEventType(domain string) (string, error) {
	if strings.Contains(domain, "*") {
		logger.Errorf(
			"Cannot generate an event type for the wildcarded domain %s."+
				"For wildcarded domains, define a custom event type with event_type"+
				"in the collection configuration file.", domain,
		)
		return "", fmt.Errorf("cannot generate event type for wildcarded domain %s", domain)
	}

	eventType := ""
	for _, s := range strings.Split(domain, ".") {
		eventType += strings.Title(s)
	}
	eventType += "Sample"

	return eventType, nil
}

// generateMetricName generates a metric name from the mbean
// This will be used if no custom metric name is defined
func generateMetricName(returnedBean string) (string, error) {

	metricName := ""
	for _, keyval := range strings.Split(returnedBean, ",") {
		val := strings.Split(keyval, "=")
		if len(val) != 2 {
			return "", fmt.Errorf("invalid selector %s", keyval)
		}
		metricName += "."
		metricName += val[1]
	}
	metricName = metricName[1:]

	return metricName, nil
}

// inferMetricType attempts to guess the metric type based
// on its ability to convert to a number
func inferMetricType(s interface{}) metric.SourceType {
	switch s.(type) {
	case int:
		return metric.GAUGE
	case float64:
		return metric.GAUGE
	case float32:
		return metric.GAUGE
	default:
		return metric.ATTRIBUTE
	}
}
