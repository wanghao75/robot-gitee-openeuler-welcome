package main

import (
	"fmt"

	libconfig "github.com/opensourceways/community-robot-lib/config"
)

type configuration struct {
	ConfigItems []botConfig `json:"config_items,omitempty"`
}

func (c *configuration) configFor(org, repo string) *botConfig {
	if c == nil {
		return nil
	}

	items := c.ConfigItems
	v := make([]libconfig.IPluginForRepo, len(items))
	for i := range items {
		v[i] = &items[i]
	}

	if i := libconfig.FindConfig(org, repo, v); i >= 0 {
		return &items[i]
	}
	return nil
}

func (c *configuration) Validate() error {
	if c == nil {
		return nil
	}

	items := c.ConfigItems
	for i := range items {
		if err := items[i].validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c *configuration) SetDefault() {
	if c == nil {
		return
	}

	Items := c.ConfigItems
	for i := range Items {
		Items[i].setDefault()
	}
}

type botConfig struct {
	libconfig.PluginForRepo
	// CommunityName displayed in welcome message
	CommunityName string `json:"community_name" required:"true"`
	// CommunityRepository name of the community's information management repository
	CommunityRepository string `json:"community_repository" required:"true"`
	// CommandLink the command help document link displayed in the welcome message
	CommandLink string `json:"command_link" required:"true"`
	// SigFilePath file path of the operation information maintenance of all
	// Special Interest Groups (SIG) in the community.
	SigFilePath string `json:"sig_file_path" required:"true"`
}

func (c *botConfig) setDefault() {
}

func (c *botConfig) validate() error {
	if c.CommunityName == "" {
		return fmt.Errorf("the community_name configuration can not be empty")
	}
	if c.CommandLink == "" {
		return fmt.Errorf("the command_link configuration can not be empty")
	}
	return c.PluginForRepo.Validate()
}
