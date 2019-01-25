// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package app

import (
	"fmt"
	"github.com/mattermost/mattermost-server/mlog"
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

func (a *App) OnboardLMSUser(userId string, lms model.LMS, launchData map[string]string) *model.AppError {
	teamSlug := lms.GetTeam(launchData)
	team, err := a.GetTeamByName(teamSlug)
	if err != nil {
		// TODO create team here
		return err
	}

	if _, err := a.GetTeamMember(team.Id, userId); err != nil {
		if _, err := a.AddTeamMember(team.Id, userId); err != nil {
			mlog.Error(fmt.Sprintf("Error occurred while adding user %s to team %s: %s", userId, team.Id, err.Error()))
		}
	}

	a.createAndJoinChannels(team.Id, lms.GetPublicChannelsToJoin(launchData), model.CHANNEL_OPEN, userId)
	a.createAndJoinChannels(team.Id, lms.GetPrivateChannelsToJoin(launchData), model.CHANNEL_PRIVATE, userId)

	return nil
}

func (a *App) createAndJoinChannels(teamId string, channels map[string]string, channelType string, userId string) {
	for slug, displayName := range channels {
		channel, err := a.GetChannelByName(slug, teamId, true)
		if err != nil {
			// channel doesnt exist
			channel = &model.Channel {
				TeamId: teamId,
				Type: channelType,
				Name: slug,
				DisplayName: displayName,
			}

			if _, err := a.CreateChannel(channel, false); err != nil {
				mlog.Error("Failed to create channel for LMS onboardin: " + err.Error())
			}
		}

		if _, err := a.AddChannelMember(userId, channel, "", "", false); err != nil {
			mlog.Error("Error occurred while adding user ID: " + userId + " to channel " + displayName + ": " + err.Error())
		}
	}
}

func (a *App) searchTeamsByName(name string) ([]*model.Team, *model.AppError) {
	if result := <-a.Srv.Store.Team().SearchByName(name); result.Err != nil {
		return nil, result.Err
	} else {
		return result.Data.([]*model.Team), nil
	}
}
