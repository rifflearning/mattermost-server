// Copyright (c) 2019-present Riff Analytics All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"errors"
	"net/http"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/shared/mlog"
	"github.com/mattermost/mattermost-server/v5/store"
)

func (a *App) GetLMSToUse(consumerKey string) model.LMS {
	for _, lms := range a.Config().LTISettings.GetKnownLMSs() {
		if lms.GetOAuthConsumerKey() == consumerKey {
			return lms
		}
	}
	return nil
}

func (a *App) OnboardLTIUser(userId string, lms model.LMS, launchData map[string]string) *model.AppError {
	teamName := lms.GetTeam(launchData)
	if err := a.addTeamMemberIfRequired(userId, teamName); err != nil {
		return err
	}

	team, err := a.GetTeamByName(teamName)
	if err != nil {
		return err
	}

	publicChannels := a.createChannelsIfRequired(team.Id, lms.GetPublicChannelsToJoin(launchData), model.CHANNEL_OPEN)
	a.joinChannelsIfRequired(userId, publicChannels)

	privateChannels := a.createChannelsIfRequired(team.Id, lms.GetPrivateChannelsToJoin(launchData), model.CHANNEL_PRIVATE)
	a.joinChannelsIfRequired(userId, privateChannels)

	return nil
}

func (a *App) PatchLTIUser(userId string, lms model.LMS, launchData map[string]string) (*model.User, *model.AppError) {
	user, err := a.GetUser(userId)
	if err != nil {
		return nil, err
	}

	user.Props[model.LTI_USER_ID_PROP_KEY] = lms.GetUserId(launchData)
	user, err = a.UpdateUser(user, false)
	if err != nil {
		return nil, err
	}

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

	if err := a.OnboardLTIUser(userId, lms, launchData); err != nil {
		return nil, err
	}

	return user, nil
}

func (a *App) SyncLTIChannels(lms model.LMS, launchData map[string]string) *model.AppError {
	teamSlug := lms.GetTeam(launchData)
	team, err := a.GetTeamByName(teamSlug)
	if err != nil {
		return err
	}

	a.syncLTIChannels(lms.GetPublicChannelsToJoin(launchData), team.Id)
	a.syncLTIChannels(lms.GetPrivateChannelsToJoin(launchData), team.Id)

	return nil
}

func (a *App) GetUserByLTI(ltiUserID string) (*model.User, *model.AppError) {
	user, err := a.Srv().Store.User().GetByLTI(ltiUserID)
	if err != nil {
		var nfErr *store.ErrNotFound
		switch {
		case errors.As(err, &nfErr):
			return nil, model.NewAppError("GetUserByLTI", MissingAccountError, nil, nfErr.Error(), http.StatusNotFound)
		default:
			return nil, model.NewAppError("GetUserByLTI", MissingAccountError, nil, err.Error(), http.StatusInternalServerError)
		}
	}
	return user, nil
}

// GetLTIUser can be used to get an LTI user by lti user id or email
func (a *App) GetLTIUser(ltiUserID, email string) *model.User {
	user, err := a.GetUserByLTI(ltiUserID)
	if err != nil {
		user, err = a.GetUserByEmail(email)
	}
	if err != nil {
		mlog.Debug("LTI User not found", mlog.String("lti_id", ltiUserID), mlog.String("email", email), mlog.Err(err))
		// return nil if the user is not found by email or LTI prop
		return nil
	}
	return user
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
				mlog.Error("Failed to create channel for LMS onboarding: "+err.Error(), mlog.Any("Channel", channel))
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
			if _, err := a.AddChannelMember(userId, channel, "", ""); err != nil {
				mlog.Error("User could not be added to channel: "+err.Error(), mlog.String("UserId", userId), mlog.String("ChannelId", channel.Id))
				continue
			}
		}
	}
}

func (a *App) addTeamMemberIfRequired(userId string, teamName string) *model.AppError {
	team, err := a.GetTeamByName(teamName)
	if err != nil {
		mlog.Error("Team to be used could not be found: "+err.Error(), mlog.String("TeamName", teamName))
		return model.NewAppError("OnboardLTIUser", "app.onboard_lms_user.team_not_found.app_error", nil, "", http.StatusInternalServerError)
	}

	if _, err := a.GetTeamMember(team.Id, userId); err != nil {
		// user is not a member of team. Adding team member
		if _, err := a.AddTeamMember(team.Id, userId); err != nil {
			mlog.Error("Error occurred while adding user to team: "+err.Error(), mlog.String("UserId", userId), mlog.String("TeamId", team.Id))
		}
	}

	return nil
}

func (a *App) syncLTIChannels(channels map[string]string, teamId string) {
	mlog.Debug("Syncing LTI channels")
	channelNames := []string{}

	for slug := range channels {
		channelNames = append(channelNames, slug)
	}

	c, err := a.GetChannelsByNames(channelNames, teamId)
	if err != nil {
		mlog.Error(err.Error())
		return
	}

	for _, channel := range c {
		// update channel if display name has changed
		if channel.DisplayName != channels[channel.Name] {
			channel.DisplayName = channels[channel.Name]
			if _, err := a.UpdateChannel(channel); err != nil {
				mlog.Error(err.Error())
			}
		}
	}
}
