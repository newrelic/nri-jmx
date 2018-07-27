package main

import (
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	yaml "gopkg.in/yaml.v2"
)

// collectionDefinitionParser is a struct to aid the automatic
// parsing of a collection yaml file
type collectionDefinitionParser struct {
	Collect []struct {
		Domain    string                 `yaml:"domain"`
		EventType string                 `yaml:"event_type"`
		Beans     []beanDefinitionParser `yaml:"beans"`
	}
}

// beanDefinitionParser is a struct to aid the automatic
// parsing of a collection yaml file
type beanDefinitionParser struct {
	Query      string        `yaml:"query"`
	Exclude    interface{}   `yaml:"exclude_regex"`
	Attributes []interface{} `yaml:"attributes"`
}

// domainDefinition is a validated and simplified
// representation of the requested collection parameters
// from a single domain
type domainDefinition struct {
	domain    string
	eventType string
	beans     []*beanRequest
}

// attributeRequest is a storage struct containing
// the information necessary to turn a JMX attribute
// into a metric
type attributeRequest struct {
	// attrRegexp is a compiled regex pattern that matches the attribute
	attrRegexp *regexp.Regexp
	metricName string
	metricType metric.SourceType
}

// beanRequest is a storage struct containing the
// information necessary to query a JMX endpoint
// and filter the results
type beanRequest struct {
	beanQuery string
	// exclude is a list of compiled regex that matches beans to exclude from collection
	exclude    []*regexp.Regexp
	attributes []*attributeRequest
}

var (
	// metricTypes maps the string used in yaml to a metric type
	metricTypes = map[string]metric.SourceType{
		"gauge":     metric.GAUGE,
		"delta":     metric.DELTA,
		"attribute": metric.ATTRIBUTE,
		"rate":      metric.RATE,
	}
)

// parseYaml reads a yaml file and parses it into a collectionDefinitionParser.
// It validates syntax only and not content
func parseYaml(filename string) (*collectionDefinitionParser, error) {
	// Read the file
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		logger.Errorf("Failed to open %s: %s", filename, err)
		return nil, err
	}

	// Parse the file
	var c collectionDefinitionParser
	if err := yaml.Unmarshal(yamlFile, &c); err != nil {
		logger.Errorf("Failed to parse collection: %s", err)
		return nil, err
	}

	return &c, nil
}

// parseCollection takes a raw collectionDefinitionParser and returns
// an array of domains containing the validated configuration
func parseCollectionDefinition(c *collectionDefinitionParser) ([]*domainDefinition, error) {

	// For each domain in the collection
	var collections []*domainDefinition
	for _, domain := range c.Collect {

		// For each bean in the domain
		var beans []*beanRequest
		for _, bean := range domain.Beans {

			// Parse the bean and add it to the domain
			newBean, err := parseBean(&bean)
			if err != nil {
				return nil, err
			}

			beans = append(beans, newBean)
		}

		// If no custom event type defined, generate an event type from the domain name
		var eventType string
		var err error
		if domain.EventType == "" {
			eventType, err = generateEventType(domain.Domain)
			if err != nil {
				return nil, err
			}
		} else {
			eventType = domain.EventType
		}
		collections = append(collections, &domainDefinition{domain: domain.Domain, eventType: eventType, beans: beans})
	}

	return collections, nil
}

func parseAttributes(rawAttributes []interface{}) ([]*attributeRequest, error) {
	var attributes []*attributeRequest
	if len(rawAttributes) == 0 {
		r, _ := createAttributeRegex(".*", false)
		attributes = []*attributeRequest{
			{
				attrRegexp: r,
				metricType: -1,
			},
		}
	} else {
		for _, attribute := range rawAttributes {
			var newAttribute *attributeRequest
			var err error
			switch a := attribute.(type) {
			case map[interface{}]interface{}:
				newAttribute, err = parseAttributeFromMap(a)
			case string:
				newAttribute, err = parseAttributeFromString(a)
			default:
				return nil, fmt.Errorf("Unable to parse attributes list %s", attribute)
			}
			if err != nil {
				return nil, err
			}
			attributes = append(attributes, newAttribute)
		}
	}

	return attributes, nil
}

func parseBean(bean *beanDefinitionParser) (*beanRequest, error) {
	attributes, err := parseAttributes(bean.Attributes)
	if err != nil {
		return nil, err
	}

	var excludePatterns []*regexp.Regexp
	if bean.Exclude != nil {
		switch b := bean.Exclude.(type) {
		case string:
			r, err := regexp.Compile(b)
			if err != nil {
				return nil, fmt.Errorf("Invalid regex pattern %s", b)
			}
			excludePatterns = append(excludePatterns, r)
		case []interface{}:
			for _, excludeString := range b {
				r, err := regexp.Compile(excludeString.(string))
				if err != nil {
					return nil, fmt.Errorf("Invalid regex pattern %s", excludeString)
				}
				excludePatterns = append(excludePatterns, r)
			}
		default:
			return nil, fmt.Errorf("Invalid format for exclude_regex")

		}
	}

	return &beanRequest{beanQuery: bean.Query, exclude: excludePatterns, attributes: attributes}, nil
}

func createAttributeRegex(attrRegex string, literal bool) (*regexp.Regexp, error) {
	var attrString string
	if literal {
		attrString = regexp.QuoteMeta(attrRegex)
	} else {
		attrString = attrRegex
	}
	r, err := regexp.Compile("attr=" + attrString + "$")
	if err != nil {
		return nil, err
	}

	return r, nil
}

func parseAttributeFromString(a string) (*attributeRequest, error) {
	attrRegexp, err := createAttributeRegex(a, true)
	if err != nil {
		return nil, fmt.Errorf("Failed to create regex pattern from attribute name %s", a)
	}

	return &attributeRequest{attrRegexp: attrRegexp}, nil
}

func parseAttributeFromMap(a map[interface{}]interface{}) (*attributeRequest, error) {
	attrName, namePresent := a["attr"]
	attrRegexpString, regexPresent := a["attr_regex"]
	var attrRegexp *regexp.Regexp
	var err error
	if !namePresent && !regexPresent {
		return nil, fmt.Errorf("must specify one of attr or attr_regex for every attribute")
	} else if namePresent && regexPresent {
		return nil, fmt.Errorf("must specify only one of attr or attr_regex")
	} else if regexPresent {
		attrRegexp, err = createAttributeRegex(attrRegexpString.(string), false)
		if err != nil {
			return nil, fmt.Errorf("failed to compile attribute regex pattern %s", attrRegexpString)
		}
	} else {
		attrRegexp, err = createAttributeRegex(attrName.(string), true)
		if err != nil {
			return nil, fmt.Errorf("failed to create regex pattern from attribute name %s", attrName.(string))
		}
	}

	metricType, err := getMetricType(a)
	if err != nil {
		return nil, err
	}
	newAttribute := &attributeRequest{
		attrRegexp: attrRegexp,
		metricType: metricType,
	}

	metricName, _ := a["metric_name"]
	if metricName != nil {
		newAttribute.metricName = metricName.(string)
	}

	return newAttribute, nil

}

func getMetricType(a map[interface{}]interface{}) (metric.SourceType, error) {
	metricTypeString, ok := a["metric_type"]
	var metricType metric.SourceType
	if !ok {
		metricType = -1 // Since metric type can't be nil, using -1 as a placeholder
	} else {
		mt, ok := metricTypes[metricTypeString.(string)]
		if !ok {
			return 0, fmt.Errorf("invalid metric type %s", metricTypeString.(string))
		}
		metricType = mt
	}

	return metricType, nil

}
