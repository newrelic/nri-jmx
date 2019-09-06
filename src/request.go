package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
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

func runCollection(collection []*domainDefinition, i *integration.Integration, host, port string) error {
	for _, domain := range collection {
		var errors []error
		for _, request := range domain.beans {
			requestString := fmt.Sprintf("%s:%s", domain.domain, request.beanQuery)
			result, err := jmxQueryFunc(requestString, args.Timeout)
			if err != nil {
				log.Error("Failed to retrieve metrics for request %s: %s", requestString, err)
				return err
			}
			if err := handleResponse(domain.eventType, request, result, i, host, port); err != nil {
				errors = append(errors, err)
			}
		}

		if len(errors) != 0 {
			log.Error("Failed to parse some responses for domain %s: %v", domain.domain, errors)
		}
	}

	return nil
}

// handleResponse takes a response, filters out the excluded beans,
// sorts the responses by domain, and passes each domain off to
// insertDomainMetrics to populate the metric list
func handleResponse(eventType string, request *beanRequest, response queryResponse, i *integration.Integration, host, port string) error {

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
		err := insertDomainMetrics(eventType, domain, beanAttrVals, request, i, host, port)
		if err != nil {
			return err
		}
	}

	return nil
}

// insertDomainMetrics akes a domain and a list of attr:value pairs,
// creates an entity and metric set for the domain, and populates the
// metric set for each attribute to be collected
func insertDomainMetrics(eventType string, domain string, beanAttrVals []*beanAttrValuePair, request *beanRequest, i *integration.Integration, host, port string) error {

	// Create an entity for the domain
	var e *integration.Entity
	var err error
	if args.LocalEntity {
		e = i.LocalEntity()
	} else {
		hostIDAttr := integration.NewIDAttribute("host", host)
		portIDAttr := integration.NewIDAttribute("port", port)
		e, err = i.Entity(domain, "jmx-domain", hostIDAttr, portIDAttr)
		if err != nil {
			return err
		}
	}

	// Create a map of bean names to metric sets
	entityMetricSets := make(map[string]*metric.Set)

	// For each bean/attribute returned from this domain
	for _, beanAttrVal := range beanAttrVals {
		// For each attribute we want to collect, check if it matches
		for _, attribute := range request.attributes {
			if attribute.attrRegexp.MatchString(beanAttrVal.beanAttr) {
				beanName, err := getBeanName(beanAttrVal.beanAttr)
				if err != nil {
					return err
				}

				// Get the metric set from the map or create it
				metricSet, err := getOrCreateMetricSet(entityMetricSets, e, request, beanName, eventType, domain)
				if err != nil {
					return err
				}

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

// getOrCreateMetricSet takes a map of bean names to metric sets and either
// returns a metric set from the map if it exists, or creates the metric set
// and adds it to the map
func getOrCreateMetricSet(entityMetricSets map[string]*metric.Set, e *integration.Entity, request *beanRequest, beanNameMatch string, eventType string, domain string) (*metric.Set, error) {

	// If the metric set exists, return it
	if ms, ok := entityMetricSets[beanNameMatch]; ok {
		return ms, nil
	}

	// Attributes in all metric sets
	attributes := []metric.Attribute{
		{Key: "query", Value: request.beanQuery},
		{Key: "domain", Value: domain},
		{Key: "host", Value: args.JmxHost},
		{Key: "bean", Value: beanNameMatch},
	}

	if !args.LocalEntity {
		nonLocalKeys := []metric.Attribute{
			{Key: "entityName", Value: "domain:" + e.Metadata.Name},
			{Key: "displayName", Value: e.Metadata.Name},
		}
		attributes = append(attributes, nonLocalKeys...)
	}

	// Add the bean keys and properties as attributes
	keyProperties, err := getKeyProperties(beanNameMatch)
	if err != nil {
		return nil, err
	}
	for key, val := range keyProperties {
		attributes = append(attributes, metric.Attribute{Key: "key:" + key, Value: val})
	}

	// Create the metric set and put it in the map
	metricSet := e.NewMetricSet(eventType, attributes...)
	entityMetricSets[beanNameMatch] = metricSet

	return metricSet, nil
}

// Inserts a metric into a metric set, generating metric names
// and metric types if unset
func insertMetric(key string, val interface{}, attribute *attributeRequest, metricSet *metric.Set) error {

	// Generate a metric name if unset
	metricName, err := func() (string, error) {
		if attribute.metricName == "" {
			metricName, err := getAttrName(key)
			if err != nil {
				return "", err
			}

			return metricName, nil
		}

		return attribute.metricName, nil
	}()

	if err != nil {
		return err
	}

	// Generate a metric type if unset
	var metricType metric.SourceType
	if attribute.metricType == -1 {
		metricType = inferMetricType(val)
	} else {
		metricType = attribute.metricType
	}

	if metricType == metric.ATTRIBUTE {
		if err := metricSet.SetMetric(metricName, fmt.Sprintf("%v", val), metricType); err != nil {
			return err
		}
	} else {
		if err := metricSet.SetMetric(metricName, val, metricType); err != nil {
			return err
		}
	}

	return nil
}

func getBeanName(beanString string) (string, error) {
	beanNameRegex := regexp.MustCompile("^(.*),attr=.*")
	beanNameMatches := beanNameRegex.FindStringSubmatch(beanString)
	if beanNameMatches == nil {
		return "", fmt.Errorf("failed to get bean name from %s", beanString)
	}

	return beanNameMatches[1], nil
}

func getAttrName(beanString string) (string, error) {
	attrNameRegex := regexp.MustCompile("^.*attr=(.*)$")
	attrNameMatches := attrNameRegex.FindStringSubmatch(beanString)
	if attrNameMatches == nil {
		return "", fmt.Errorf("failed to get attr name from %s", beanString)
	}

	return attrNameMatches[1], nil
}

func getKeyProperties(keyProperties string) (keyPropertiesMap map[string]string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to parse properties %s", keyProperties)
		}
	}()
	keyPropertiesMap = make(map[string]string)

	i := 0
	tokenStart := 0
	key := ""
	for i < len(keyProperties) {
		// Find the key
		if keyProperties[i] == '=' {
			key = keyProperties[tokenStart:i]
			i++
			// Find the value
			if keyProperties[i] == '"' { // value is quoted
				i++
				tokenStart := i
				for i < len(keyProperties) {
					if keyProperties[i] == '"' && keyProperties[i-1] != '\\' {
						keyPropertiesMap[key] = keyProperties[tokenStart:i]
						i += 2
						break
					}
					i++
				}
			} else { // value is not quoted
				tokenStart = i
				for { // search for first comma
					if i == len(keyProperties) || keyProperties[i] == ',' {
						keyPropertiesMap[key] = keyProperties[tokenStart:i]
						i++
						break
					}
					i++
				}
			}
			tokenStart = i
		} else {
			i++
		}
	}

	return keyPropertiesMap, nil

}

// Convenience function to split the domain:query string
// into domain and query
func splitBeanName(bean string) (string, string, error) {
	domainQuery := strings.SplitN(bean, ":", 2)
	if len(domainQuery) != 2 {
		return "", "", fmt.Errorf("invalid domain:bean string %s", bean)
	}
	return domainQuery[0], domainQuery[1], nil
}

// generateEventType generates an event type from a domain string.
// The resulting event type will be used if no custom event type has been defined.
func generateEventType(domain string) (string, error) {
	if strings.Contains(domain, "*") {
		log.Error(
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
