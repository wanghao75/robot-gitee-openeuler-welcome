package main

import (
	"fmt"
	"strings"

	"github.com/opensourceways/community-robot-lib/config"
	framework "github.com/opensourceways/community-robot-lib/robot-gitee-framework"
	"github.com/opensourceways/community-robot-lib/utils"
	sdk "github.com/opensourceways/go-gitee/gitee"
	"github.com/sirupsen/logrus"
)

const (
	botName        = "welcome"
	welcomeMessage = `
Hi ***%s***, welcome to the %s Community.
I'm the Bot here serving you. You can find the instructions on how to interact with me at **[Here](%s)**.
If you have any questions, please contact the SIG: [%s](https://gitee.com/openeuler/community/tree/master/sig/%s), and any of the maintainers: @%s`
)

type iClient interface {
	CreatePRComment(owner, repo string, number int32, comment string) error
	CreateIssueComment(owner, repo string, number string, comment string) error
	GetPathContent(org, repo, path, ref string) (sdk.Content, error)
	CreateRepoLabel(org, repo, label, color string) error
	GetRepoLabels(owner, repo string) ([]sdk.Label, error)
	AddPRLabel(org, repo string, number int32, label string) error
	AddIssueLabel(org, repo, number, label string) error
	ListCollaborators(org, repo string) ([]sdk.ProjectMember, error)
	GetDirectoryTree(org, repo, sha string, recursive int32) (sdk.Tree, error)
}

func newRobot(cli iClient) *robot {
	return &robot{cli: cli}
}

type robot struct {
	cli iClient
}

func (bot *robot) NewConfig() config.Config {
	return &configuration{}
}

func (bot *robot) getConfig(cfg config.Config, org, repo string) (*botConfig, error) {
	c, ok := cfg.(*configuration)
	if !ok {
		return nil, fmt.Errorf("can't convert to configuration")
	}

	if bc := c.configFor(org, repo); bc != nil {
		return bc, nil
	}

	return nil, fmt.Errorf("no config for this repo:%s/%s", org, repo)
}

func (bot *robot) RegisterEventHandler(p framework.HandlerRegitster) {
	p.RegisterIssueHandler(bot.handleIssueEvent)
	p.RegisterPullRequestHandler(bot.handlePREvent)
}

func (bot *robot) handlePREvent(e *sdk.PullRequestEvent, pc config.Config, log *logrus.Entry) error {
	if sdk.GetPullRequestAction(e) != sdk.ActionOpen {
		return nil
	}

	org, repo := e.GetOrgRepo()
	cfg, err := bot.getConfig(pc, org, repo)
	if err != nil {
		return err
	}

	number := e.GetPRNumber()

	return bot.handle(
		org, repo, e.GetPRAuthor(), cfg, log,

		func(c string) error {
			return bot.cli.CreatePRComment(org, repo, number, c)
		},

		func(label string) error {
			return bot.cli.AddPRLabel(org, repo, number, label)
		},
	)
}

func (bot *robot) handleIssueEvent(e *sdk.IssueEvent, pc config.Config, log *logrus.Entry) error {
	if sdk.ActionOpen != e.GetAction() {
		return nil
	}

	org, repo := e.GetOrgRepo()
	cfg, err := bot.getConfig(pc, org, repo)
	if err != nil {
		return err
	}

	author := e.GetIssueAuthor()
	number := e.GetIssueNumber()

	return bot.handle(
		org, repo, author, cfg, log,

		func(c string) error {
			return bot.cli.CreateIssueComment(org, repo, number, c)
		},

		func(label string) error {
			return bot.cli.AddIssueLabel(org, repo, number, label)
		},
	)
}

func (bot *robot) handle(
	org, repo, author string,
	cfg *botConfig, log *logrus.Entry,
	addMsg, addLabel func(string) error,
) error {
	sigName, comment, err := bot.genComment(org, repo, author, cfg)
	if err != nil {
		return err
	}

	mErr := utils.NewMultiErrors()

	if err := addMsg(comment); err != nil {
		mErr.AddError(err)
	}

	label := fmt.Sprintf("sig/%s", sigName)

	if err := bot.createLabelIfNeed(org, repo, label); err != nil {
		log.Errorf("create repo label:%s, err:%s", label, err.Error())
	}

	if err := addLabel(label); err != nil {
		mErr.AddError(err)
	}

	return mErr.Err()
}

func (bot robot) genComment(org, repo, author string, cfg *botConfig) (string, string, error) {
	sigName, err := bot.getSigOfRepo(org, repo, cfg)
	if err != nil {
		return "", "", err
	}

	if sigName == "" {
		return "", "", fmt.Errorf("cant get sig name of repo: %s/%s", org, repo)
	}

	maintainers, err := bot.getMaintainers(org, repo)
	if err != nil {
		return "", "", err
	}

	return sigName, fmt.Sprintf(
		welcomeMessage, author, cfg.CommunityName, cfg.CommandLink,
		sigName, sigName, strings.Join(maintainers, " , @"),
	), nil
}

func (bot *robot) getMaintainers(org, repo string) ([]string, error) {
	v, err := bot.cli.ListCollaborators(org, repo)
	if err != nil {
		return nil, err
	}

	r := make([]string, 0, len(v))
	for i := range v {
		p := v[i].Permissions
		if p != nil && (p.Admin || p.Push) {
			r = append(r, v[i].Login)
		}
	}
	return r, nil
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
