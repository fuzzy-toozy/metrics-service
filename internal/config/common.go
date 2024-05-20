// Package config Duration option for use in configuration.
package config

import (
	"encoding/json"
	"os"
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
	intDuration, err := strconv.ParseUint(flagValue, 10, 64)
	if err == nil {
		o.D = time.Duration(intDuration) * time.Second
		return nil
	}

	d, err := time.ParseDuration(flagValue)
	if err != nil {
		return err
	}
	o.D = d

	return nil
}
func (o *DurationOption) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.String())
}

func (o *DurationOption) UnmarshalJSON(data []byte) error {
	var durationString string
	if err := json.Unmarshal(data, &durationString); err != nil {
		return err
	}

	intDuration, err := strconv.ParseUint(durationString, 10, 64)
	if err == nil {
		o.D = time.Duration(intDuration) * time.Second
		return nil
	}

	duration, err := time.ParseDuration(durationString)
	if err != nil {
		return err
	}
	o.D = duration
	return nil
}

func ParseConfigFile(path string, data any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	err = json.NewDecoder(f).Decode(data)
	if err != nil {
		return err
	}

	return nil
}
