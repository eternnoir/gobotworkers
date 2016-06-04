package sitechecker

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	log "github.com/Sirupsen/logrus"
	"github.com/eternnoir/gobot"
	"github.com/eternnoir/gobot/payload"
	"net/http"
	"regexp"
	"text/template"
	"time"
)

const WORKER_ID string = "SITECHECKER"

func init() {
	gobot.RegisterWorker(WORKER_ID, &SiteChecker{})
}

type SiteChecker struct {
	bot             *gobot.Gobot
	sites           map[string]string
	CheckInterval   time.Duration
	MessageTemplate *template.Template
	Command         string
	checking        bool
}

type NotifyPayload struct {
	SiteName   string
	SiteUrl    string
	StatusCode string
}

func (w *SiteChecker) Init(bot *gobot.Gobot) error {
	w.bot = bot
	var conf Config
	if _, err := toml.DecodeFile(bot.ConfigPath+"/sitechecker.toml", &conf); err != nil {
		return err
	}
	log.Infof("%s Load cconfig %#v", WORKER_ID, conf)
	if conf.SiteUrl == nil {
		errors.New(WORKER_ID + " sites can not be nil.")
	}
	w.sites = conf.SiteUrl
	w.CheckInterval = conf.CheckInterval.Duration

	temlateString := "{{.SiteName}} is dead!!! Status {{.StatusCode}} {{.SiteUrl}}"
	if conf.MessageTemplate != "" {
		temlateString = conf.MessageTemplate
	}

	tmpl, err := template.New("sitechecker").Parse(temlateString)
	if err != nil {
		return err
	}
	w.MessageTemplate = tmpl

	command := "sitestatus"
	if conf.Command == "" {
		command = conf.Command
	}
	w.Command = command
	w.checking = false
	return nil
}

func (w *SiteChecker) Process(gobot *gobot.Gobot, message *payload.Message) error {
	if !w.checking {
		go w.run()
	}
	return nil
}

func (w *SiteChecker) run() {
	w.checking = true
	for {
		for sitename, url := range w.sites {
			go w.checkAndSendResult(sitename, url)
		}
		time.Sleep(w.CheckInterval)
	}
}

func (w *SiteChecker) checkAndSendResult(siteName, url string) {
	resp, err := CheckStatus(url)
	if err != nil {
		log.Debugf("Check %s %s Fail, error %s", siteName, url, err)
		w.bot.Send(fmt.Sprintf("Check site %s Fail. Error %s", siteName, err))
		return
	}
	log.Debugf("Check %s %s , result %s", siteName, url, resp.Status)
	if resp.StatusCode == http.StatusOK {
		return
	}
	var doc bytes.Buffer
	err = w.MessageTemplate.Execute(&doc, NotifyPayload{SiteName: siteName, SiteUrl: url, StatusCode: resp.Status})
	if err != nil {
		log.Errorf("Conver template error. %s", err)
	}
	msg := doc.String()
	log.Infof("[%s] Send message. %s", WORKER_ID, msg)
	w.bot.Send(msg)
}

func CheckStatus(url string) (*http.Response, error) {
	log.Debugf("Check url %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	return resp, err
}

func CheckIsUrl(url string) bool {
	r, err := regexp.Compile(`[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,6}\b([-a-zA-Z0-9@:%_\+.~#?&//=]*)`)
	if err != nil {
		return false
	}
	return r.MatchString(url)
}
