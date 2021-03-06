package models

import "errors"

type StatsConfig struct {
	Org      string `json:"org"`
	Team     string `json:"team"`
	Owner    string `json:"owner"`
	ApiToken string `json:"apiToken"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	File     string `json:"file"`
}

func (c StatsConfig) Validate() error {
	if len(c.ApiToken) == 0 {
		return errors.New("api token is missing")
	}
	if len(c.Org) == 0 && len(c.Owner) == 0 {
		return errors.New("organization is missing")
	}
	if len(c.Org) > 0 && len(c.Owner) > 0 {
		return errors.New("organization and owner cannot be set at the same time")
	}
	if len(c.Team) == 0 && len(c.Owner) == 0 {
		return errors.New("team is missing")
	}
	if len(c.Team) > 0 && len(c.Owner) > 0 {
		return errors.New("team and owner cannot be set at the same time")
	}
	if len(c.Owner) == 0 && len(c.Org) == 0 {
		return errors.New("Owner is missing")
	}
	if c.Start >= 0 {
		return errors.New("start is invalid")
	}
	if c.End > 0 {
		return errors.New("end is invalid")
	}
	if len(c.File) == 0 {
		return errors.New("file is missing")
	}
	return nil
}
