package model

import (
	"net/http"
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
	Name  string
	Type  string
	OAuth LMSOAuthSettings
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
		Email:     launchData["lis_person_contact_email_primary"],
		Username:  launchData["lis_person_sourcedid"],
		FirstName: launchData["lis_person_name_given"],
		LastName:  launchData["lis_person_name_family"],
		Position:  launchData["roles"],
		Password:  password,
		Props: StringMap{
			"lti_user_id": launchData["custom_user_id"],
		},
	}
}
