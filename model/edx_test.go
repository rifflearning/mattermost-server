// Copyright (c) 2019-present Riff Analytics All Rights Reserved.
// See LICENSE.txt for license information.

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildUser(t *testing.T) {
	lms := &EdxLMS{
		Name:                "LMS_Name",
		Type:                "edx",
		OAuthConsumerKey:    "consumer_key",
		OAuthConsumerSecret: "consumer_secret",
		Teams: map[string]string{
			"context_id_1": "team-1",
		},
		PersonalChannels: EdxPersonalChannels{
			Type: "type",
			ChannelList: map[string]EdxChannel{
				"plg": {
					IdProperty:   "plg_id_property",
					NameProperty: "plg_name_property",
				},
				"capstone": {
					IdProperty:   "capstone_id_property",
					NameProperty: "capstone_name_property",
				},
			},
		},
		DefaultChannels: map[string]EdxDefaultChannel{},
	}

	launchData := map[string]string{
		"context_id":                       "context_id_1",
		"lis_person_contact_email_primary": "foo@example.com",
		"lis_person_sourcedid":             "lti_username",
		"lis_person_name_given":            "first",
		"lis_person_name_family":           "last",
		"roles":                            "lti_roles",
		"custom_user_id":                   "user_5",
	}

	user, err := lms.BuildUser(launchData, "password")
	assert.Nil(t, err)

	assert.Equal(t, "foo@example.com", user.Email)
	assert.Equal(t, "lti_username", user.Username)
	assert.Equal(t, "first", user.FirstName)
	assert.Equal(t, "last", user.LastName)
	assert.Equal(t, "lti_roles", user.Position)
	assert.Equal(t, "password", user.Password)
	assert.Contains(t, user.Props, "lti_user_id")
	assert.Equal(t, "user_5", user.Props["lti_user_id"])

	// Test BuildUser w/o a custom_user_id field in the launchData
	// should compose an lti_user_id w/ the consumer key and user's email
	delete(launchData, "custom_user_id")
	user, err = lms.BuildUser(launchData, "password")
	assert.Nil(t, err)

	assert.Contains(t, user.Props, "lti_user_id")
	assert.Equal(t, "consumer_key:foo@example.com", user.Props["lti_user_id"])
}

func TestGetPrivateChannelsToJoin(t *testing.T) {
	lms := &EdxLMS{
		Name:                "LMS_Name",
		Type:                "edx",
		OAuthConsumerKey:    "consumer_key",
		OAuthConsumerSecret: "consumer_secret",
		Teams: map[string]string{
			"context_id_1": "team-1",
		},
		PersonalChannels: EdxPersonalChannels{
			Type: "type",
			ChannelList: map[string]EdxChannel{
				"plg": {
					IdProperty:   "plg_id_property",
					NameProperty: "plg_name_property",
				},
				"capstone": {
					IdProperty:   "capstone_id_property",
					NameProperty: "capstone_name_property",
				},
			},
		},
		DefaultChannels: map[string]EdxDefaultChannel{},
	}

	plgChannelId := "plg_slug"
	plgChannelSlug := GetLMSChannelSlug("plg", plgChannelId)
	capstoneChannelId := "capstone_slug"
	capstoneChannelSlug := GetLMSChannelSlug("capstone", capstoneChannelId)

	launchData := map[string]string{
		"context_id":                       "context_id_1",
		"plg_id_property":                  plgChannelId,
		"plg_name_property":                "Plg DN",
		"capstone_id_property":             capstoneChannelId,
		"capstone_name_property":           "Capstone DN",
		"lis_person_contact_email_primary": "foo@example.com",
		"lis_person_sourcedid":             "lti_username",
		"roles":                            "lti_roles",
		"custom_user_id":                   "lti_user_id",
	}

	channels := lms.GetPrivateChannelsToJoin(launchData)

	assert.Contains(t, channels, plgChannelSlug)
	assert.Contains(t, channels, capstoneChannelSlug)
	assert.Equal(t, "Plg DN", channels[plgChannelSlug])
	assert.Equal(t, "Capstone DN", channels[capstoneChannelSlug])
}

func TestGetChannel(t *testing.T) {
	lms := &EdxLMS{
		Name:                "LMS_Name",
		Type:                "edx",
		OAuthConsumerKey:    "consumer_key",
		OAuthConsumerSecret: "consumer_secret",
		Teams: map[string]string{
			"context_id_1": "team-1",
		},
		PersonalChannels: EdxPersonalChannels{
			Type: "type",
			ChannelList: map[string]EdxChannel{
				"plg": {
					IdProperty:   "plg_id_property",
					NameProperty: "plg_name_property",
				},
				"capstone": {
					IdProperty:   "capstone_id_property",
					NameProperty: "capstone_name_property",
				},
			},
		},
		DefaultChannels: map[string]EdxDefaultChannel{},
	}

	plgChannelId := "plg_slug"
	plgChannelSlug := GetLMSChannelSlug("plg", plgChannelId)
	capstoneChannelId := "capstone_slug"
	capstoneChannelSlug := GetLMSChannelSlug("capstone", capstoneChannelId)

	launchData := map[string]string{
		"context_id":                       "context_id_1",
		"custom_channel_redirect":          "lookup:plg",
		"plg_id_property":                  plgChannelId,
		"plg_name_property":                "Plg DN",
		"capstone_id_property":             capstoneChannelId,
		"capstone_name_property":           "Capstone DN",
		"lis_person_contact_email_primary": "foo@example.com",
		"lis_person_sourcedid":             "lti_username",
		"roles":                            "lti_roles",
		"custom_user_id":                   "lti_user_id",
	}

	channelSlug, err := lms.GetChannel(launchData)
	assert.Nil(t, err)
	assert.Equal(t, plgChannelSlug, channelSlug)

	launchData["custom_channel_redirect"] = "lookup:capstone"
	channelSlug, err = lms.GetChannel(launchData)
	assert.Nil(t, err)
	assert.Equal(t, capstoneChannelSlug, channelSlug)
}

func TestSyncUser(t *testing.T) {
	lms := &EdxLMS{
		Name:                "LMS_Name",
		Type:                "edx",
		OAuthConsumerKey:    "consumer_key",
		OAuthConsumerSecret: "consumer_secret",
		Teams: map[string]string{
			"context_id_1": "team-1",
		},
		PersonalChannels: EdxPersonalChannels{
			Type: "type",
			ChannelList: map[string]EdxChannel{
				"plg": {
					IdProperty:   "plg_id_property",
					NameProperty: "plg_name_property",
				},
				"capstone": {
					IdProperty:   "capstone_id_property",
					NameProperty: "capstone_name_property",
				},
			},
		},
		DefaultChannels: map[string]EdxDefaultChannel{},
	}

	launchData := map[string]string{
		"context_id":                       "context_id_1",
		"lis_person_contact_email_primary": "foo@example.com",
		"lis_person_sourcedid":             "lti_username",
		"lis_person_name_given":            "first",
		"lis_person_name_family":           "last",
		"roles":                            "lti_roles",
		"custom_user_id":                   "user_5",
	}

	user := &User{}

	user = lms.SyncUser(user, launchData)
	assert.Equal(t, "foo@example.com", user.Email)
	assert.Equal(t, "lti_username", user.Username)
	assert.Equal(t, "lti_roles", user.Position)
	assert.Contains(t, user.Props, "lti_user_id")
	assert.Equal(t, "user_5", user.Props["lti_user_id"])
}
