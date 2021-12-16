package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nrjmx/gojmx"
)

// Client interface for JMX client.
type Client interface {
	Connect(config *gojmx.JMXConfig) error
	Disconnect() error
	Query(mBeanDomain, mBeanMetric string) ([]*gojmx.JMXAttribute, error)
	QueryMBean(mBeanNamePattern string) ([]*gojmx.JMXAttribute, error)
}

// jmxClient will handle the connection to JMX.
type jmxClient struct {
	conn *gojmx.Client
}

// NewJMXClient returns new JMXClient.
func NewJMXClient() Client {
	return &jmxClient{
		conn: gojmx.NewClient(context.Background()),
	}
}

// Connect preforms connect to JMX endpoint.
func (c *jmxClient) Connect(config *gojmx.JMXConfig) error {
	_, err := c.conn.Open(config)
	if jmxErr, ok := gojmx.IsJMXError(err); ok {
		return errors.New(jmxErr.Message)
	} else if jmxConnErr, ok := gojmx.IsJMXConnectionError(err); ok {
		return errors.New(jmxConnErr.Message)
	}
	return err
}

// Disconnect closes the JMX endpoint connection.
func (c *jmxClient) Disconnect() error {
	return c.conn.Close()
}

// Query the JMX endpoint for JMXAttributes
func (c *jmxClient) Query(mBeanDomain, mBeanMetric string) ([]*gojmx.JMXAttribute, error) {
	mBeanNamePattern := fmt.Sprintf("%s:%s", mBeanDomain, mBeanMetric)
	return c.QueryMBean(mBeanNamePattern)
}

// Query the JMX endpoint for JMXAttributes
func (c *jmxClient) QueryMBean(mBeanNamePattern string) ([]*gojmx.JMXAttribute, error) {
	var result []*gojmx.JMXAttribute

	mBeanNames, err := c.conn.GetMBeanNames(mBeanNamePattern)
	if jmxErr := handleError(mBeanNamePattern, err); jmxErr != nil {
		return nil, jmxErr
	}

	for _, mBeanName := range mBeanNames {
		mBeanAttrNames, err := c.conn.GetMBeanAttrNames(mBeanName)
		if jmxErr := handleError(mBeanNamePattern, err); jmxErr != nil {
			return nil, jmxErr
		}

		for _, mBeanAttrName := range mBeanAttrNames {
			jmxAttributes, err := c.conn.GetMBeanAttrs(mBeanName, mBeanAttrName)
			if jmxErr := handleError(mBeanNamePattern, err); jmxErr != nil {
				return nil, jmxErr
			}

			result = append(result, jmxAttributes...)
		}
	}
	return result, nil
}

func handleError(mBeanNamePattern string, err error) error {
	if err == nil {
		return nil
	}
	if jmxErr, ok := gojmx.IsJMXError(err); ok {
		log.Debug("error while querying mBean pattern: '%s', error message: %s, error cause: %s, stacktrace: %q",
			mBeanNamePattern,
			jmxErr.Message,
			jmxErr.CauseMessage,
			jmxErr.Stacktrace,
		)
		return nil
	} else if jmxConnErr, ok := gojmx.IsJMXConnectionError(err); ok {
		return fmt.Errorf("collection failed, error: %v", jmxConnErr.Message)
	}
	return fmt.Errorf("collection failed, error: %v", err)
}
