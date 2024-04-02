// Duration option for use in configuration.
package config

import (
	"strconv"
	"time"
)

type DurationOption struct {
	D time.Duration
}

func (o *DurationOption) String() string {
	return o.D.String()
}

func (o *DurationOption) Set(flagValue string) error {
	intD, err := strconv.ParseInt(flagValue, 10, 64)
	if err != nil {
		return err
	}
	o.D = time.Duration(intD) * time.Second
	return nil
}
