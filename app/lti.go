// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package app

import (
	"fmt"
	"net/http"

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

func (a *App) OnboardLTIUser(userId string, lms model.LMS, launchData map[string]string, joinChannels bool) *model.AppError {
	teamName := lms.GetTeam(launchData)
	if err := a.addTeamMemberIfRequired(userId, teamName); err != nil {
		return err
	}

	team, err := a.GetTeamByName(teamName)
	if err != nil {
		return err
	}

	publicChannels := a.createChannelsIfRequired(team.Id, lms.GetPublicChannelsToJoin(launchData), model.CHANNEL_OPEN)
	privateChannels := a.createChannelsIfRequired(team.Id, lms.GetPrivateChannelsToJoin(launchData), model.CHANNEL_PRIVATE)

	if joinChannels {
		a.joinChannelsIfRequired(userId, publicChannels)
		a.joinChannelsIfRequired(userId, privateChannels)
	}

	return nil
}

func (a *App) PatchLTIUser(userId string, lms model.LMS, launchData map[string]string) (*model.User, *model.AppError) {
	user, err := a.GetUser(userId)
	if err != nil {
		return nil, err
	}

	user.Props[model.LTI_USER_ID_PROP_KEY] = lms.GetUserId(launchData)
	return user, nil
}

func (a *App) SyncLTIUser(userId string, lms model.LMS, launchData map[string]string) (*model.User, *model.AppError) {
	user, err := a.GetUser(userId)
	if err != nil {
		return nil, err
	}

	lms.SyncUser(user, launchData)
	user, err = a.UpdateUser(user, false)
	if err != nil {
		return nil, err
	}

	// TODO: confirm if we need to re-join channels or not
	if err := a.OnboardLTIUser(userId, lms, launchData, false); err != nil {
		return nil, err
	}

	return user, nil
}

func (a *App) createChannelsIfRequired(teamId string, channels map[string]string, channelType string) model.ChannelList {
	var channelList model.ChannelList
	for slug, displayName := range channels {
		channel, err := a.GetChannelByName(slug, teamId, true)
		if err != nil {
			// channel doesnt exist, create it
			channel = &model.Channel{
				TeamId:      teamId,
				Type:        channelType,
				Name:        slug,
				DisplayName: displayName,
			}

			channel, err = a.CreateChannel(channel, false)
			if err != nil {
				mlog.Error("Failed to create channel for LMS onboarding: " + err.Error())
				continue
			}
		}

		channelList = append(channelList, channel)
	}

	return channelList
}

func (a *App) joinChannelsIfRequired(userId string, channels model.ChannelList) {
	for _, channel := range channels {
		_, err := a.GetChannelMember(channel.Id, userId)
		if err != nil {
			// channel member doesn't exist
			// add user to channel
			if _, err := a.AddChannelMember(userId, channel, "", "", false); err != nil {
				mlog.Error(fmt.Sprintf("User with ID %s could not be added to chanel with ID %s. Error: %s", userId, channel.Id, err.Error()))
				continue
			}
		}
	}
}

func (a *App) addTeamMemberIfRequired(userId string, teamName string) *model.AppError {
	team, err := a.GetTeamByName(teamName)
	if err != nil {
		mlog.Error(fmt.Sprintf("Team to be used: %s could not be found: %s", teamName, err.Error()))
		return model.NewAppError("OnboardLTIUser", "app.onboard_lms_user.team_not_found.app_error", nil, "", http.StatusInternalServerError)
	}

	if _, err := a.GetTeamMember(team.Id, userId); err != nil {
		// user is not a member of team. Adding team member
		if _, err := a.AddTeamMember(team.Id, userId); err != nil {
			mlog.Error(fmt.Sprintf("Error occurred while adding user %s to team %s: %s", userId, team.Id, err.Error()))
		}
	}

	return nil
}
