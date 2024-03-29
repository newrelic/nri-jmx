/*
 * Copyright 2022 New Relic Corporation. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"errors"

	"github.com/newrelic/nrjmx/gojmx"
)

var (
	ErrJMXCollection = errors.New("JMX collection failed")
	ErrConnectionErr = errors.New("JMX connection failed")
)

// Client interface for JMX connection.
type Client interface {
	Open(config *gojmx.JMXConfig) (*gojmx.Client, error)
	Close() error
	QueryMBeanAttributes(mBeanNamePattern string, mBeanAttributeName ...string) ([]*gojmx.AttributeResponse, error)
}
