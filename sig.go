package main

import (
	"encoding/base64"

	"sigs.k8s.io/yaml"

	"github.com/opensourceways/robot-gitee-openeuler-welcome/community"
)

func (bot *robot) getSigOfRepo(repo string, cfg *botConfig) (string, error) {
	c, err := bot.loadFile(cfg.sigFile)
	if err != nil {
		return "", err
	}

	s := new(community.Sigs)
	if err := decodeYamlFile(c, s); err != nil {
		return "", err
	}

	if err := s.Validate(); err != nil {
		return "", err
	}

	return s.GetSig(repo), nil
}

func (bot *robot) loadFile(f fileOfRepo) (string, error) {
	c, err := bot.cli.GetPathContent(f.org, f.repo, f.path, f.branch)
	if err != nil {
		return "", err
	}

	return c.Content, nil
}

func decodeYamlFile(content string, v interface{}) error {
	c, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(c, v)
}
