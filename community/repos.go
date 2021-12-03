package community

import (
	"fmt"
	"strings"
)

type Sigs struct {
	Items []Sig `json:"sigs,omitempty"`
}

func (s *Sigs) Validate() error {
	if s == nil {
		return fmt.Errorf("empty sigs")
	}

	for i := range s.Items {
		if err := s.Items[i].validate(); err != nil {
			return fmt.Errorf("validate %d sig, err:%s", i, err)
		}
	}

	return nil
}

func (s *Sigs) GetSig(repo string) string {
	if s == nil {
		return ""
	}

	for i := range s.Items {
		if s.Items[i].hasRepo(repo) {
			return s.Items[i].Name
		}
	}

	return ""
}

type Sig struct {
	Name         string   `json:"name" required:"true"`
	Repositories []string `json:"repositories,omitempty"`
}

func (s *Sig) hasRepo(repo string) bool {
	if s == nil {
		return false
	}

	repo = "/" + repo
	for _, item := range s.Repositories {
		if strings.HasSuffix(item, repo) {
			return true
		}
	}
	return false
}

func (s *Sig) validate() error {
	if s.Name == "" {
		return fmt.Errorf("missing sig name")
	}

	return nil
}
