package sitechecker

import (
	"bytes"
	"errors"
	"github.com/BurntSushi/toml"
	log "github.com/Sirupsen/logrus"
	"github.com/eternnoir/gobot"
	"github.com/eternnoir/gobot/payload"
	"net/http"
	"regexp"
	"strings"
	"text/template"
	"time"
)

const WORKER_ID string = "SITECHECKER"

func init() {
	gobot.RegisterWorker(WORKER_ID, &SiteChecker{})
}

type SiteChecker struct {
	bot                   *gobot.Gobot
	sites                 map[string]string
	CheckInterval         time.Duration
	FailMessageTemplate   *template.Template
	StatusMessageTemplate *template.Template
	Command               string
	checking              bool
	chatroom              string
}

type NotifyPayload struct {
	SiteName   string
	SiteUrl    string
	StatusCode string
	Msg        string
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

	failtemlateString := "{{.SiteName}} is dead!!! Status {{.StatusCode}} {{.SiteUrl}}"
	if conf.FailMessageTemplate != "" {
		failtemlateString = conf.FailMessageTemplate
	}

	tmpl, err := template.New("sitechecker").Parse(failtemlateString)
	if err != nil {
		return err
	}
	w.FailMessageTemplate = tmpl

	statusTemplateString := "{{.SiteName}} Status {{.StatusCode}} {{.SiteUrl}}"
	if conf.StatusMessageTemplate != "" {
		statusTemplateString = conf.StatusMessageTemplate
	}

	stmpl, ers := template.New("sitecheckerstatus").Parse(statusTemplateString)
	if ers != nil {
		return err
	}
	w.StatusMessageTemplate = stmpl

	command := "sitestatus"
	if conf.Command != "" {
		command = conf.Command
	}
	w.Command = command
	w.chatroom = conf.ChatRoom
	w.checking = false
	return nil
}

func (w *SiteChecker) Process(gobot *gobot.Gobot, message *payload.Message) error {
	if !w.checking {
		go w.run()
	}
	log.Infof("[%s] Get new message %s", WORKER_ID, message.Text)

	if strings.Contains(message.Text, w.Command) {
		for sitename, url := range w.sites {
			go w.checkAndSendResult(sitename, url, w.StatusMessageTemplate, false, message)
		}
	}

	return nil
}

func (w *SiteChecker) run() {
	w.checking = true
	for {
		for sitename, url := range w.sites {
			go w.checkAndSendResult(sitename, url, w.FailMessageTemplate, true, nil)
		}
		time.Sleep(w.CheckInterval)
	}
}

func (w *SiteChecker) checkAndSendResult(siteName, url string, templ *template.Template, skipsuccess bool, originMessage *payload.Message) {
	resp, err := CheckStatus(url)
	if err != nil {
		log.Debugf("Check %s %s Fail, error %s", siteName, url, err)
		msg, err := w.GetMsg(templ, NotifyPayload{SiteName: siteName, SiteUrl: url, Msg: err.Error()})
		if err != nil {
			log.Error(err)
			return
		}
		log.Infof("[%s] Send message. %s", WORKER_ID, msg)
		w.sendMessage(msg, originMessage)
		return
	}
	log.Debugf("Check %s %s , result %s", siteName, url, resp.Status)
	if resp.StatusCode == http.StatusOK && skipsuccess {
		return
	}
	msg, err := w.GetMsg(templ, NotifyPayload{SiteName: siteName, SiteUrl: url, StatusCode: resp.Status})
	if err != nil {
		log.Error(err)
		return
	}
	log.Infof("[%s] Send message. %s", WORKER_ID, msg)
	w.sendMessage(msg, originMessage)
}

func (w *SiteChecker) GetMsg(template *template.Template, payload NotifyPayload) (string, error) {
	var doc bytes.Buffer
	err := template.Execute(&doc, payload)
	if err != nil {
		log.Errorf("Conver template error. %s", err)
		return "", err
	}
	msg := doc.String()
	return msg, err
}

func (w *SiteChecker) sendMessage(msg string, originMessage *payload.Message) {
	if originMessage == nil {
		w.bot.SendToChat(msg, w.chatroom)
	} else {
		w.bot.Reply(originMessage, msg)
	}
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
