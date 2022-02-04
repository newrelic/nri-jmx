package main

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/newrelic/infra-integrations-sdk/data/attribute"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nrjmx/gojmx"
)

var (
	ErrNoDataForPattern = errors.New("empty data for pattern")
)

// beanAttrValue is a convenience struct that
// contains the fully qualified bean name + attribute
// tag and also contains the value associated with
// that attribute. It facilitates passing the attribute
// and its value between functions easily
type beanAttrValue struct {
	beanAttr    string
	attrRequest *attributeRequest
	value       interface{}
}

func runCollection(collection []*domainDefinition, i *integration.Integration, client Client, host, port string) error {
	for _, domain := range collection {
		var handlingErrs []error

		for _, request := range domain.beans {
			response, err := client.QueryMBeanAttributes(fmt.Sprintf("%s:%s", domain.domain, request.beanQuery))
			if jmxErr, ok := gojmx.IsJMXError(err); ok {
				handlingErrs = append(handlingErrs, fmt.Errorf("%w, Pattern: %s:%s, error: %v", ErrNoDataForPattern, domain.domain, request.beanQuery, jmxErr))
				continue
			} else if jmxConnErr, ok := gojmx.IsJMXConnectionError(err); ok {
				return fmt.Errorf("%w, error: %v", ErrConnectionErr, jmxConnErr.Message)
			} else if err != nil {
				return fmt.Errorf("%w, error: %v", ErrJMXCollection, err)
			}

			if len(response) == 0 {
				handlingErrs = append(handlingErrs, fmt.Errorf("%w, Pattern: %s:%s", ErrNoDataForPattern, domain.domain, request.beanQuery))
				continue
			}

			if errs := handleResponse(domain, request, response, i, host, port); errs != nil {
				handlingErrs = append(handlingErrs, errs...)
			}
		}
		for i, err := range handlingErrs {
			if i == 0 {
				log.Error("Failed to parse some responses for domain %s", domain.domain)
			}
			log.Error("can't parse response: %v", err)
		}
	}
	return nil
}

// matchRequest will return tha part of the config that requested the attribute.
func matchRequest(jmxAttr *gojmx.AttributeResponse, request *beanRequest) *attributeRequest {
	if jmxAttr == nil || request == nil {
		return nil
	}

	for _, pattern := range request.exclude {
		if pattern.MatchString(jmxAttr.Name) {
			return nil
		}
	}

	// For each attribute we want to collect, check if it matches
	for _, attribute := range request.attributes {
		if attribute.attrRegexp.MatchString(jmxAttr.Name) {
			// We want to insert the metric just once. Doesn't matter if it matched other request.
			return attribute
		}
	}

	return nil
}

// handleResponse takes a response, filters out the excluded beans,
// sorts the responses by domain, and passes each domain off to
// insertDomainMetrics to populate the metric list
func handleResponse(domain *domainDefinition, request *beanRequest, response []*gojmx.AttributeResponse, i *integration.Integration, host, port string) (handlingErrs []error) {
	// If there are multiple domains, we have to create an entity for each
	// Create a map with domain as the key that returns query/value
	domainsMap := make(map[string][]*beanAttrValue)
	for _, attribute := range response {
		attrRequest := matchRequest(attribute, request)
		// Attribute was not requested by config or was filtered.
		if attrRequest == nil {
			continue
		}

		if attribute.ResponseType == gojmx.ResponseTypeErr {
			log.Warn("Failed to process attribute for query: %s status: %s", request.beanQuery, attribute.StatusMsg)
			continue
		}

		domainName, beanAttr, err := splitBeanName(attribute.Name)
		if err != nil {
			handlingErrs = append(handlingErrs, err)
			continue
		}

		domainsMap[domainName] = append(domainsMap[domainName], &beanAttrValue{
			beanAttr:    beanAttr,
			attrRequest: attrRequest,
			value:       attribute.GetValue(),
		})
	}

	// For each domain, create an entity and a metric set
	for domainName, beanAttrVals := range domainsMap {
		err := insertDomainMetrics(domain.eventType, domainName, beanAttrVals, request, i, host, port)
		if err != nil {
			handlingErrs = append(handlingErrs, err)
			continue
		}
	}
	return
}

// insertDomainMetrics akes a domain and a list of attr:value pairs,
// creates an entity and metric set for the domain, and populates the
// metric set for each attribute to be collected
func insertDomainMetrics(eventType string, domain string, beanAttrVals []*beanAttrValue, request *beanRequest, i *integration.Integration, host, port string) error {
	// Create an entity for the domain
	var e *integration.Entity
	var err error
	switch {
	case args.RemoteMonitoring:
		url := net.JoinHostPort(host, port)
		if args.ConnectionURL != "" {
			url = getConnectionURLSAP(args.ConnectionURL)
		}
		e, err = newRemoteEntity(domain, url, i)
		if err != nil {
			return err
		}
	case args.LocalEntity:
		e = i.LocalEntity()
	default:
		// create task for consistency with remote_monitoring
		if args.ConnectionURL != "" {
			host, port = getConnectionURLHostPort(args.ConnectionURL)
		}

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
		beanName, err := getBeanName(beanAttrVal.beanAttr)
		if err != nil {
			return err
		}

		// Query the metric set from the map or create it
		metricSet, err := getOrCreateMetricSet(entityMetricSets, e, request, beanName, eventType, domain)
		if err != nil {
			return err
		}

		// If we want to collect the metric, populate the metric list
		if err := insertMetric(beanAttrVal.beanAttr, beanAttrVal.value, beanAttrVal.attrRequest, metricSet); err != nil {
			return err
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
	attributes := []attribute.Attribute{
		{Key: "query", Value: request.beanQuery},
		{Key: "domain", Value: domain},
		{Key: "host", Value: args.JmxHost},
		{Key: "bean", Value: beanNameMatch},
	}

	if !args.LocalEntity {
		nonLocalKeys := []attribute.Attribute{
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
		attributes = append(attributes, attribute.Attribute{Key: "key:" + key, Value: val})
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
	case int, int32, int64, float32, float64:
		return metric.GAUGE
	default:
		return metric.ATTRIBUTE
	}
}

func newRemoteEntity(domain, suffix string, i *integration.Integration) (*integration.Entity, error) {
	return i.Entity(fmt.Sprintf("%s:%s", domain, suffix), "jmx-domain")
}

// getConnectionURLSAP extracts last part that describes connection string,
// in case that it can't extract SAP part, will return full connection URL
// ref: https://docs.oracle.com/javase/7/docs/api/javax/management/remote/JMXServiceURL.html
func getConnectionURLSAP(connectionURL string) string {
	r := strings.Split(connectionURL, "://")
	const countOfURLSegments = 3
	if len(r) != countOfURLSegments {
		return connectionURL
	}
	return r[len(r)-1]
}

func getConnectionURLHostPort(connectionURL string) (string, string) {
	const hostAndPortCount = 2
	connectionSAP := getConnectionURLSAP(connectionURL)
	hostParts := strings.Split(connectionSAP, "/")
	if len(hostParts) == 0 {
		return "", ""
	}
	hostAndPort := strings.Split(hostParts[0], ":")
	if len(hostAndPort) != hostAndPortCount {
		return "", ""
	}
	return hostAndPort[0], hostAndPort[1]
}
