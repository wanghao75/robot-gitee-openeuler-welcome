package main

import (
	"strings"
)

func (bot *robot) getSigOfRepo(org, repo string, cfg *botConfig) (string, error) {
	sigName, err := bot.findSigName(org, repo, cfg, true)
	if err != nil {
		return sigName, err
	}

	return sigName, nil
}

func (bot *robot) listAllFilesOfRepo(cfg *botConfig) (map[string]string, error) {
	trees, err := bot.cli.GetDirectoryTree(cfg.CommunityName, cfg.CommunityRepo, cfg.Branch, 1)
	if err != nil || len(trees.Tree) == 0 {
		return nil, err
	}

	r := make(map[string]string)

	for i := range trees.Tree {
		item := &trees.Tree[i]
		if strings.Count(item.Path, "/") == 4 {
			r[item.Path] = strings.Split(item.Path, "/")[1]
		}
	}

	return r, nil
}

func (bot *robot) findSigName(org, repo string, cfg *botConfig, needRefreshTree bool) (sigName string, err error) {
	if len(cfg.reposSig) == 0 {
		files, err := bot.listAllFilesOfRepo(cfg)
		if err != nil {
			return "", err
		}

		cfg.reposSig = files
	}

	for i := range cfg.reposSig {
		if strings.Contains(i, org) && strings.Contains(i, repo) {
			sigName = cfg.reposSig[i]
			needRefreshTree = false
			break
		} else {
			continue
		}
	}

	if needRefreshTree {
		files, err := bot.listAllFilesOfRepo(cfg)
		if err != nil {
			return "", err
		}

		cfg.reposSig = files

		for i := range cfg.reposSig {
			if strings.Contains(i, org) && strings.Contains(i, repo) {
				sigName = cfg.reposSig[i]
				break
			} else {
				continue
			}
		}
	}

	return sigName, nil
}
