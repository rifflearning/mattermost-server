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
	CourseType   string
	NameProperty string
	IDProperty   string
}

type EdxUserChannelsSettings struct {
	Type        string
	ChannelList []EdxChannel
}

type EdxLMS struct {
	Name         string
	Type         string
	OAuth        LMSOAuthSettings
	UserChannels EdxUserChannelsSettings
}

func (e *EdxLMS) GetName() string {
	return e.Name
}

func (e *EdxLMS) GetType() string {
	return e.Type
}

func (e *EdxLMS) GetOAuth() LMSOAuthSettings {
	return e.OAuth
}

func (e *EdxLMS) GetValidateLTIRequest() bool {
	return true
}

func (e *EdxLMS) ValidateLTIRequest(url string, request *http.Request) bool {
	return baseValidateLTIRequest(e.OAuth.ConsumerSecret, e.OAuth.ConsumerKey, url, request)
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
			"lti_user_id": launchData[launchDataLTIUserIdKey],
		},
	}
}
