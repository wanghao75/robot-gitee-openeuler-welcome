package main

import (
	"encoding/base64"
	"fmt"
	"strings"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	libconfig "github.com/opensourceways/community-robot-lib/config"
	"github.com/opensourceways/community-robot-lib/giteeclient"
	libplugin "github.com/opensourceways/community-robot-lib/giteeplugin"
	"github.com/opensourceways/community-robot-lib/utils"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const (
	botName        = "welcome"
	welcomeMessage = `Hi ***%s***, welcome to the %s Community.
I'm the Bot here serving you. You can find the instructions on how to interact with me at
<%s>.
If you have any questions, please contact the SIG: [%s](https://gitee.com/openeuler/community/tree/master/sig/%s),and any of the maintainers: %s`
)

type iClient interface {
	CreatePRComment(owner, repo string, number int32, comment string) error
	CreateIssueComment(owner, repo string, number string, comment string) error
	GetPathContent(org, repo, path, ref string) (sdk.Content, error)
	CreateRepoLabel(org, repo, label, color string) error
	GetRepoLabels(owner, repo string) ([]sdk.Label, error)
	AddPRLabel(org, repo string, number int32, label string) error
	AddIssueLabel(org, repo, number, label string) error
}

func newRobot(cli iClient) *robot {
	return &robot{cli: cli}
}

type robot struct {
	cli iClient
}

func (bot *robot) NewPluginConfig() libconfig.PluginConfig {
	return &configuration{}
}

func (bot *robot) getConfig(cfg libconfig.PluginConfig, org, repo string) (*botConfig, error) {
	c, ok := cfg.(*configuration)
	if !ok {
		return nil, fmt.Errorf("can't convert to configuration")
	}
	if bc := c.configFor(org, repo); bc != nil {
		return bc, nil
	}
	return nil, fmt.Errorf("no config for this repo:%s/%s", org, repo)
}

func (bot *robot) RegisterEventHandler(p libplugin.HandlerRegitster) {
	p.RegisterIssueHandler(bot.handleIssueEvent)
	p.RegisterPullRequestHandler(bot.handlePREvent)
}

func (bot *robot) handlePREvent(e *sdk.PullRequestEvent, pc libconfig.PluginConfig, log *logrus.Entry) error {
	action := giteeclient.GetPullRequestAction(e)
	if action != giteeclient.PRActionOpened {
		return nil
	}

	pr := giteeclient.GetPRInfoByPREvent(e)
	cfg, err := bot.getConfig(pc, pr.Org, pr.Repo)
	if err != nil {
		return err
	}

	addMsg := func(comment string) error {
		return bot.cli.CreatePRComment(pr.Org, pr.Repo, pr.Number, comment)
	}
	addLabel := func(label string) error {
		return bot.cli.AddPRLabel(pr.Org, pr.Repo, pr.Number, label)
	}

	return bot.handle(pr.Org, pr.Repo, pr.Author, addMsg, addLabel, cfg, log)
}

func (bot *robot) handleIssueEvent(e *sdk.IssueEvent, pc libconfig.PluginConfig, log *logrus.Entry) error {
	ew := giteeclient.NewIssueEventWrapper(e)
	if giteeclient.StatusOpen != ew.GetAction() {
		return nil
	}

	org, repo := ew.GetOrgRep()
	cfg, err := bot.getConfig(pc, org, repo)
	if err != nil {
		return err
	}

	author := ew.GetIssueAuthor()
	number := ew.GetIssueNumber()
	addMsg := func(comment string) error {
		return bot.cli.CreateIssueComment(org, repo, number, comment)
	}
	addLabel := func(label string) error {
		return bot.cli.AddIssueLabel(org, repo, number, label)
	}

	return bot.handle(org, repo, author, addMsg, addLabel, cfg, log)
}

func (bot *robot) handle(
	org, repo, author string,
	addMsg, addLabel func(string) error,
	cfg *botConfig, log *logrus.Entry,
) error {
	sigName := bot.repoSigName(org, repo, cfg, log)
	if sigName == "" {
		return fmt.Errorf("cant get sig name by %s/%s", org, repo)
	}

	mErr := utils.NewMultiErrors()
	if comment := bot.genWelcomeMsg(author, sigName, cfg, log); comment != "" {
		mErr.AddError(addMsg(comment))
	}

	label := fmt.Sprintf("sig/%s", sigName)
	if err := bot.createLabelIfNeed(org, repo, label); err != nil {
		log.WithError(err).Errorf("create repo label: %s", label)
	}

	mErr.AddError(addLabel(label))

	return mErr.Err()
}

func (bot robot) genWelcomeMsg(author, sigName string, cfg *botConfig, log *logrus.Entry) string {
	v, err := bot.getMaintainers(sigName, cfg)
	if err != nil {
		log.Error(err)
		return ""
	}

	maintainerMsg := ""
	if len(v) > 0 {
		maintainerMsg = fmt.Sprintf("**@%s**", strings.Join(v, "** ,**@"))
	}

	return fmt.Sprintf(welcomeMessage, author, cfg.CommunityName, cfg.CommandLink, sigName, sigName, maintainerMsg)
}

func (bot robot) repoSigName(org, repo string, cfg *botConfig, log *logrus.Entry) string {
	c, err := bot.getPathContent(cfg.CommunityName, cfg.CommunityRepository, cfg.SigFilePath)
	if err != nil {
		log.Error(err)
		return ""
	}

	// intercept the sig configuration item string containing a repository full path from the sig yaml file
	s := string(c)
	keyOfName := "- name: "
	path := fmt.Sprintf("%s/%s", org, repo)
	repoSigConfig := interceptString(s, keyOfName, path)
	if repoSigConfig == "" {
		return ""
	}

	// intercept the sig name
	keyOfRepos := "repositories:"
	s = interceptString(repoSigConfig, keyOfName, keyOfRepos)
	if s == "" {
		return ""
	}

	s = strings.TrimPrefix(s, keyOfName)
	s = strings.TrimSuffix(s, keyOfRepos)
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "")

	return s
}

func (bot *robot) getMaintainers(sig string, cfg *botConfig) ([]string, error) {
	path := fmt.Sprintf("sig/%s/OWNERS", sig)
	c, err := bot.getPathContent(cfg.CommunityName, cfg.CommunityRepository, path)
	if err != nil {
		return nil, err
	}

	var m struct {
		Maintainers []string `yaml:"maintainers"`
	}
	if err = yaml.Unmarshal(c, &m); err != nil {
		return nil, err
	}

	return m.Maintainers, nil
}

func (bot *robot) getPathContent(owner, repo, path string) ([]byte, error) {
	content, err := bot.cli.GetPathContent(owner, repo, path, "master")
	if err != nil {
		return nil, err
	}

	c, err := base64.StdEncoding.DecodeString(content.Content)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (bot *robot) createLabelIfNeed(org, repo, label string) error {
	repoLabels, err := bot.cli.GetRepoLabels(org, repo)
	if err != nil {
		return err
	}

	for _, v := range repoLabels {
		if v.Name == label {
			return nil
		}
	}

	return bot.cli.CreateRepoLabel(org, repo, label, "")
}

// interceptString intercept the substring between the last matching `start` and the first matching `end` in a string.
// for example: enter abab12389898 ab 98 will return ab123898".
func interceptString(s, start, end string) string {
	if s == "" || start == "" || end == "" {
		return s
	}

	eIdx := strings.Index(s, end)
	if eIdx == -1 {
		return ""
	}

	eIdx += len(end)
	sIdx := strings.LastIndex(s[:eIdx], start)
	if eIdx == -1 {
		return ""
	}

	return s[sIdx:eIdx]
}
