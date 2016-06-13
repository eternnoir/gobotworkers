package gitlabmr

import (
	"time"
)

type Config struct {
	NotifyTemplate   string
	Command          string
	ResponseTemplate string
	Url              string
	Token            string
	NotifyChat       string
	Projects         []string
	PollingInterval  duration
	NotifyInterval   duration
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}
