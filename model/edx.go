// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package model

import (
	"net/http"
)

const (
	launchDataEmailKey     = "lis_person_contact_email_primary"
	launchDataUsernameKey  = "lis_person_sourcedid"
	launchDataFirstNameKey = "lis_person_name_given"
	launchDataLastNameKey  = "lis_person_name_family"
	launchDataPositionKey  = "roles"
	launchDataLTIUserIdKey = "custom_user_id"
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

func (e *EdxLMS) ValidateLTIRequest(url string, request *http.Request) bool {
	return baseValidateLTIRequest(e.OAuthConsumerSecret, e.OAuthConsumerKey, url, request)
}

func (e *EdxLMS) BuildUser(launchData map[string]string, password string) *User {
	return &User{
		Email:     launchData[launchDataEmailKey],
		Username:  transformLTIUsername(launchData[launchDataUsernameKey]),
		FirstName: launchData[launchDataFirstNameKey],
		LastName:  launchData[launchDataLastNameKey],
		Position:  launchData[launchDataPositionKey],
		Password:  password,
		Props: StringMap{
			LTI_USER_ID_PROP_KEY: launchData[launchDataLTIUserIdKey],
		},
	}
}

func (e *EdxLMS) GetTeam(launchData map[string]string) string {
	contextId := launchData["context_id"]
	return e.Teams[contextId]
}

func (e *EdxLMS) GetPublicChannelsToJoin(launchData map[string]string) map[string]string {
	// TODO check if need to join default channels if MM experimental default channel doesn't works
	return map[string]string{}
}

func (e *EdxLMS) GetPrivateChannelsToJoin(launchData map[string]string) map[string]string {
	channels := map[string]string{}

	for _, channelConfig := range e.PersonalChannels.ChannelList {
		channelDisplayName := launchData[channelConfig.NameProperty]
		channelSlug := launchData[channelConfig.IdProperty]
		channels[channelSlug] = channelDisplayName
	}

	return channels
}
