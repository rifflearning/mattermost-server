// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package model

import (
	"fmt"
	"net/http"
	"strings"
)

const (
	launchDataEmailKey           = "lis_person_contact_email_primary"
	launchDataUsernameKey        = "lis_person_sourcedid"
	launchDataFirstNameKey       = "lis_person_name_given"
	launchDataLastNameKey        = "lis_person_name_family"
	launchDataPositionKey        = "roles"
	launchDataLTIUserIdKey       = "custom_user_id"
	launchDataChannelRedirectKey = "custom_channel_redirect"
	launchDataFullNameKey        = "lis_person_name_full"

	redirectChannelLookupKeyword = "lookup"
)

type EdxChannel struct {
	IdProperty   string
	NameProperty string
}

type EdxPersonalChannels struct {
	Type        string
	ChannelList map[string]EdxChannel
}

type EdxDefaultChannel struct {
	Name        string
	DisplayName string
}

type EdxLMS struct {
	Name                string
	Type                string
	OAuthConsumerKey    string
	OAuthConsumerSecret string
	Teams               map[string]string

	PersonalChannels EdxPersonalChannels
	DefaultChannels  map[string]EdxDefaultChannel
}

func (e *EdxLMS) GetEmail(launchData map[string]string) string {
	return launchData[launchDataEmailKey]
}

func (e *EdxLMS) GetName() string {
	return e.Name
}

func (e *EdxLMS) GetType() string {
	return e.Type
}

func (e *EdxLMS) GetOAuthConsumerKey() string {
	return e.OAuthConsumerKey
}

func (e *EdxLMS) GetOAuthConsumerSecret() string {
	return e.OAuthConsumerSecret
}

func (e *EdxLMS) GetUserId(launchData map[string]string) string {
	return launchData[launchDataLTIUserIdKey]
}

func (e *EdxLMS) ValidateLTIRequest(url string, request *http.Request) bool {
	return baseValidateLTIRequest(e.OAuthConsumerSecret, e.OAuthConsumerKey, url, request)
}

func (e *EdxLMS) BuildUser(launchData map[string]string, password string) (*User, *AppError) {
	//checking if all required fields are present

	if launchData[launchDataEmailKey] == "" {
		return nil, NewAppError("Edx_BuildUser", "edx.build_user.email_missing", nil, "", http.StatusBadRequest)
	}

	if launchData[launchDataUsernameKey] == "" {
		return nil, NewAppError("Edx_BuildUser", "edx.build_user.username_missing", nil, "", http.StatusBadRequest)
	}

	props := StringMap{}
	props[LTI_USER_ID_PROP_KEY] = e.GetUserId(launchData)

	if props[LTI_USER_ID_PROP_KEY] == "" {
		return nil, NewAppError("Edx_BuildUser", "edx.build_user.lti_user_id_missing", nil, "", http.StatusBadRequest)
	}

	firstName := strings.Trim(launchData[launchDataFirstNameKey], " ")
	lastName := strings.Trim(launchData[launchDataLastNameKey], " ")

	if firstName == "" || lastName == "" {
		name := strings.SplitN(strings.Trim(launchData[launchDataFullNameKey], " "), " ", 2)
		if firstName == "" && len(name) > 0 {
			firstName = name[0]
		}

		if lastName == "" && len(name) > 1 {
			lastName = name[1]
		}
	}

	if firstName == "" {
		firstName = launchData[launchDataUsernameKey]
	}

	user := &User{
		FirstName: firstName,
		LastName:  lastName,
		Email:     launchData[launchDataEmailKey],
		Username:  transformLTIUsername(launchData[launchDataUsernameKey]),
		Position:  launchData[launchDataPositionKey],
		Password:  password,
		Props:     props,
	}

	return user, nil
}

func (e *EdxLMS) GetTeam(launchData map[string]string) string {
	contextId := launchData["context_id"]
	return e.Teams[contextId]
}

func (e *EdxLMS) GetPublicChannelsToJoin(launchData map[string]string) map[string]string {
	return map[string]string{}
}

func (e *EdxLMS) GetPrivateChannelsToJoin(launchData map[string]string) map[string]string {
	channels := map[string]string{}

	for personalChannelName, channelConfig := range e.PersonalChannels.ChannelList {
		channelDisplayName := launchData[channelConfig.NameProperty]
		channelSlug := fmt.Sprintf(
			"%s-%s", personalChannelName, launchData[channelConfig.IdProperty],
		)[0:CHANNEL_NAME_UI_MAX_LENGTH]

		if channelDisplayName != "" && channelSlug != "" {
			channels[channelSlug] = channelDisplayName
		}
	}

	return channels
}

func (e *EdxLMS) GetChannel(launchData map[string]string) (string, *AppError) {
	customChannelRedirect, ok := launchData[launchDataChannelRedirectKey]
	if !ok {
		return "", nil
	}

	var channelSlug string

	components := strings.Split(customChannelRedirect, ":")
	if len(components) == 1 {
		channelSlug = components[0]
	} else if components[0] == redirectChannelLookupKeyword {
		edxChannel, ok := e.PersonalChannels.ChannelList[components[1]]
		if !ok {
			return "", NewAppError("GetChannel", "get_channel.redirect_lookup_channel.not_found", nil, "", http.StatusBadRequest)
		}

		channelSlug = fmt.Sprintf("%s-%s", components[1], launchData[edxChannel.IdProperty])
	} else {

	}

	return channelSlug, nil
}

func (e *EdxLMS) SyncUser(user *User, launchData map[string]string) *User {
	if launchData[launchDataEmailKey] != "" {
		user.Email = launchData[launchDataEmailKey]
	}

	if launchData[launchDataUsernameKey] != "" {
		user.Username = transformLTIUsername(launchData[launchDataUsernameKey])
	}

	if launchData[launchDataPositionKey] != "" {
		user.Position = launchData[launchDataPositionKey]
	}

	if user.Props == nil {
		user.Props = StringMap{}
	}

	user.Props[LTI_USER_ID_PROP_KEY] = e.GetUserId(launchData)
	return user
}
