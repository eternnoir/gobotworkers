package gitlabmr

import (
	"bytes"
	"fmt"
	"github.com/BurntSushi/toml"
	log "github.com/Sirupsen/logrus"
	"github.com/eternnoir/gmrn/apis"
	"github.com/eternnoir/gmrn/notifier"
	"github.com/eternnoir/gobot"
	"github.com/eternnoir/gobot/payload"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const WORKER_ID string = "GITLABMERGEREQUEST"

func init() {
	gobot.RegisterWorker(WORKER_ID, &MrWorker{})
}

type MrWorker struct {
	runner           *notifier.NotifyRunner
	gitlabNotifier   *notifier.GitLabNotifier
	chatroom         string
	command          string
	NotifyTemplate   *template.Template
	PollingInterval  time.Duration
	ResponseTemplate *template.Template
	mergeRequestMap  map[string]bool
	bot              *gobot.Gobot
}

func (w *MrWorker) Init(bot *gobot.Gobot) error {
	var conf Config
	if _, err := toml.DecodeFile(bot.ConfigPath+"/gitlabmr.toml", &conf); err != nil {
		log.Errorf("[%s] Load config fail. %s", WORKER_ID, err)
		return err
	}
	log.Infof("%s Load config %#v", WORKER_ID, conf)
	ntmpl, err := template.New("noti").Parse(conf.NotifyTemplate)
	if err != nil {
		log.Error(err)
		return err
	}
	w.NotifyTemplate = ntmpl

	rtmpl, er := template.New("resp").Parse(conf.ResponseTemplate)
	if er != nil {
		log.Error(err)
		return err
	}
	w.ResponseTemplate = rtmpl

	command := "mergerequest"
	if conf.Command != "" {
		command = conf.Command
	}

	w.command = command
	w.PollingInterval = conf.PollingInterval.Duration
	w.chatroom = conf.NotifyChat

	w.gitlabNotifier = notifier.InitGitLabNotifier(conf.Url, conf.Token, conf.Projects, w.PollingInterval, conf.NotifyInterval.Duration)
	w.gitlabNotifier.CheckProjects()
	w.bot = bot
	go w.RunNotify()
	return nil
}

func (w *MrWorker) RunNotify() {
	w.mergeRequestMap = map[string]bool{}
	for {
		log.Infof("[%s] Start check new mergerequests.", WORKER_ID)
		w.notifyNewMergequest()
		time.Sleep(w.PollingInterval)
	}
}

func (w *MrWorker) notifyNewMergequest() {
	mrs, err := w.gitlabNotifier.GetAllProjectsMr()
	if err != nil {
		log.Errorf("[%s] GetAllPrjectsMr Fail. %s", WORKER_ID, err)
		return
	}
	for _, mr := range mrs {
		uuname := strconv.Itoa(int(mr.Iid)) + "-" + strconv.Itoa(int(mr.ProjectId))
		if _, ok := w.mergeRequestMap[uuname]; !ok {
			w.mergeRequestMap[uuname] = true
			log.Infof("[%s] Get new MergeRequest %#v", WORKER_ID, mr)
			w.notiMr(w.bot, mr)
		}
	}

	log.Debugf("[%s] Total cache mrs %#v", WORKER_ID, w.mergeRequestMap)
}

func (w *MrWorker) notiMr(gobot *gobot.Gobot, mr *apis.MergeRequest) {
	err := mr.GetProjectInfo(w.gitlabNotifier.Api)
	if err != nil {
		log.Errorf("[%s] Get project info Fail. %#v. %s", WORKER_ID, mr, err.Error())
		return
	}
	var doc bytes.Buffer
	err = w.NotifyTemplate.Execute(&doc, mr)
	if err != nil {
		log.Errorf("Convert ntoitemplate error. %s.%#v", err, mr)
		return
	}
	msg := doc.String()
	gobot.SendToChat(msg, w.chatroom)
}

func (w *MrWorker) Process(gobot *gobot.Gobot, message *payload.Message) error {
	log.Debugf("[%s] Get new message %#v", WORKER_ID, message)
	if message == nil {
		return fmt.Errorf("[%s] Get null message.", WORKER_ID)
	}

	if strings.Contains(w.command, message.Text) {
		log.Infof("[%s] Process message %s", WORKER_ID, message.Text)
		gobot.Reply(message, fmt.Sprintf("Checking MergeRequest for %s. Please wait", message.FromUser.Name))
		w.processNewGetMRMsg(gobot, message)
	}

	return nil
}

func (w *MrWorker) processNewGetMRMsg(gobot *gobot.Gobot, message *payload.Message) error {
	log.Debugf("[%s] Try get merge requests.", WORKER_ID)
	mrs, err := w.gitlabNotifier.GetAllProjectsMr()
	if err != nil {
		log.Error(err)
		return err
	}
	log.Debugf("[%s] Get %d mergequests", WORKER_ID, len(mrs))
	w.sendMatchedResponse(gobot, mrs, message)

	return nil
}

func (w *MrWorker) sendMatchedResponse(gobot *gobot.Gobot, allMergeRequests []*apis.MergeRequest, message *payload.Message) {
	log.Debugf("[%s] Start to find merge requests for %s", WORKER_ID, message.FromUser.Name)
	fromUserName := message.FromUser.Name
	mrCount := 0
	for _, mr := range allMergeRequests {
		if mr.Assignee == nil {
			continue
		}

		if mr.Assignee.UserName == fromUserName {
			log.Debugf("[%s] Find MergeRequest for %s. %#v", WORKER_ID, fromUserName, mr)
			w.replyRespMrToUser(gobot, mr, message)
			mrCount = mrCount + 1
		}
	}

	if mrCount == 0 {
		gobot.Reply(message, "Congratulation!! (=￣ω￣=) You don't have any MergeRequest.")
	} else {
		gobot.Reply(message, fmt.Sprintf("@%s. You have %d Merge Requests.", fromUserName, mrCount))
	}
}

func (w *MrWorker) replyRespMrToUser(gobot *gobot.Gobot, mr *apis.MergeRequest, message *payload.Message) {
	err := mr.GetProjectInfo(w.gitlabNotifier.Api)
	if err != nil {
		log.Errorf("[%s] Get project info Fail. %#v. %s", WORKER_ID, mr, err.Error())
		return
	}
	var doc bytes.Buffer
	err = w.ResponseTemplate.Execute(&doc, mr)
	if err != nil {
		log.Errorf("Conver response template error. %s.%#v", err, mr)
		return
	}
	msg := doc.String()
	gobot.Reply(message, msg)
}
