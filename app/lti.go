package app

import (
	"github.com/mattermost/mattermost-server/model"
)

func (a *App) GetLMSToUse(consumerKey string) model.LMS {
	for _, lms := range a.Config().LTISettings.GetKnownLMSs() {
		if lms.GetOAuth().ConsumerKey == consumerKey  {
			return lms
		}
	}
	return nil
}
