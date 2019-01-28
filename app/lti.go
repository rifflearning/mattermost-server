// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package app

import (
	"github.com/mattermost/mattermost-server/model"
)

func (a *App) GetLMSToUse(consumerKey string) model.LMS {
	for _, lms := range a.Config().LTISettings.GetKnownLMSs() {
		if lms.GetOAuthConsumerKey() == consumerKey {
			return lms
		}
	}
	return nil
}

func (a *App) PatchLTIUser(userId string, ltiUserId string) (*model.User, *model.AppError) {
	user, err := a.GetUser(userId)
	if err != nil {
		return nil, err
	}

	user.Props[model.LTI_USER_ID_PROP_KEY] = ltiUserId
	user, err = a.UpdateUser(user, false)
	if err != nil {
		return nil, err
	}

	return user, nil
}
