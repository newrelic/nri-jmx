package main

import (
	"errors"

	"github.com/newrelic/nrjmx/gojmx"
)

var (
	ErrJMXCollection = errors.New("JMX collection failed")
	ErrConnectionErr = errors.New("JMX format failed")
)

// Client interface for JMX format.
type Client interface {
	Open(config *gojmx.JMXConfig) (*gojmx.Client, error)
	Close() error
	QueryMBean(mBeanNamePattern string) (gojmx.QueryResponse, error)
}
