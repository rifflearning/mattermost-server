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

type EdxDefaultChannelMapping map[string]EdxDefaultChannel

type EdxTeamMapping [] struct {
	ContextId    string
	TeamName string
}

type EdxLMS struct {
	Type                string
	OAuthConsumerKey    string
	OAuthConsumerSecret string
	Teams               EdxTeamMapping

	PersonalChannels EdxPersonalChannels
	DefaultChannels  EdxDefaultChannelMapping
}

func (e *EdxLMS) GetName() string {
	return "Appsembler"
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

func (e *EdxLMS) GetValidateLTIRequest() bool {
	return true
}

func (e *EdxLMS) ValidateLTIRequest(url string, request *http.Request) bool {
	return baseValidateLTIRequest(e.OAuthConsumerSecret, e.OAuthConsumerKey, url, request)
}

func (e *EdxLMS) BuildUser(launchData map[string]string, password string) *User {
	return &User{
		Email:     launchData[launchDataEmailKey],
		Username:  launchData[launchDataUsernameKey],
		FirstName: launchData[launchDataFirstNameKey],
		LastName:  launchData[launchDataLastNameKey],
		Position:  launchData[launchDataPositionKey],
		Password:  password,
		Props: StringMap{
			LTI_USER_ID_PROP_KEY: launchData[launchDataLTIUserIdKey],
		},
	}
}
