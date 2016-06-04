package sitechecker

import (
	"time"
)

type Config struct {
	SiteUrl         map[string]string
	MessageTemplate string
	CheckInterval   duration
	Command         string
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}
