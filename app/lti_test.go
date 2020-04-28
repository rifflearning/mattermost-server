// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package app

import (
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/stretchr/testify/assert"
)

func TestApp_OnboardLMSUser(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()

	user := th.BasicUser
	lms := &model.EdxLMS{
		Name:                "LMS_Name",
		Type:                "edx",
		OAuthConsumerKey:    "consumer_key",
		OAuthConsumerSecret: "consumer_secret",
		Teams: map[string]string{
			"context_id_1": th.BasicTeam.Name,
		},
		PersonalChannels: model.EdxPersonalChannels{
			Type: "type",
			ChannelList: map[string]model.EdxChannel{
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
		DefaultChannels: map[string]model.EdxDefaultChannel{},
	}

	launchData := map[string]string{
		"context_id":        "context_id_1",
		"plg_id_property":   "channel_slug",
		"plg_name_property": "Channel Display Name",
	}

	err := th.App.OnboardLTIUser(user.Id, lms, launchData)
	assert.Nil(t, err)

	lmsChannel, err := th.App.GetChannelByName("channel_slug", th.BasicTeam.Id, false)
	assert.Nil(t, err)
	assert.NotNil(t, lmsChannel)
	assert.Equal(t, "channel_slug", lmsChannel.Name)
	assert.Equal(t, "Channel Display Name", lmsChannel.DisplayName)
	assert.Equal(t, model.CHANNEL_PRIVATE, lmsChannel.Type)

	member, err := th.App.GetChannelMember(lmsChannel.Id, user.Id)
	assert.Nil(t, err)
	assert.NotNil(t, member)
	assert.Equal(t, user.Id, member.UserId)
	assert.Equal(t, lmsChannel.Id, member.ChannelId)

}

func TestApp_PatchLTIUser(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()

	user := th.BasicUser
	_, ok := user.Props[model.LTI_USER_ID_PROP_KEY]
	assert.False(t, ok)

	lms := &model.EdxLMS{}
	launchData := map[string]string{
		"custom_user_id": "abc123",
	}

	patchedUser, err := th.App.PatchLTIUser(user.Id, lms, launchData)
	assert.Nil(t, err)
	assert.Equal(t, "abc123", patchedUser.Props[model.LTI_USER_ID_PROP_KEY])

	// testing on already patched user
	patchedUser, err = th.App.PatchLTIUser(user.Id, lms, launchData)
	assert.Nil(t, err)
	assert.Equal(t, "abc123", patchedUser.Props[model.LTI_USER_ID_PROP_KEY])
}

func TestApp_SyncLTIUser(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()

	lms := &model.EdxLMS{
		Name:                "LMS_Name",
		Type:                "edx",
		OAuthConsumerKey:    "consumer_key",
		OAuthConsumerSecret: "consumer_secret",
		Teams: map[string]string{
			"context_id_1": th.BasicTeam.Name,
		},
		PersonalChannels: model.EdxPersonalChannels{
			Type: "type",
			ChannelList: map[string]model.EdxChannel{
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
		DefaultChannels: map[string]model.EdxDefaultChannel{},
	}

	launchData := map[string]string{
		"context_id":                       "context_id_1",
		"plg_id_property":                  "channel_slug",
		"plg_name_property":                "Channel Display Name",
		"lis_person_contact_email_primary": "foo@example.com",
		"lis_person_sourcedid":             "lti_username",
		"roles":                            "lti_roles",
		"custom_user_id":                   "lti_user_id",
	}

	user := th.BasicUser
	syncedUser, err := th.App.SyncLTIUser(user.Id, lms, launchData)
	assert.Nil(t, err)
	assert.Equal(t, "foo@example.com", syncedUser.Email)
	assert.Equal(t, "lti_username", syncedUser.Username)
	assert.Equal(t, "lti_roles", syncedUser.Position)
	assert.Equal(t, "lti_user_id", syncedUser.Props[model.LTI_USER_ID_PROP_KEY])
}

func TestApp_SyncLTIChannels(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()

	lms := &model.EdxLMS{
		Name:                "LMS_Name",
		Type:                "edx",
		OAuthConsumerKey:    "consumer_key",
		OAuthConsumerSecret: "consumer_secret",
		Teams: map[string]string{
			"context_id_1": th.BasicTeam.Name,
		},
		PersonalChannels: model.EdxPersonalChannels{
			Type: "type",
			ChannelList: map[string]model.EdxChannel{
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
		DefaultChannels: map[string]model.EdxDefaultChannel{},
	}

	plgChannelId := "plg_slug"
	plgChannelSlug := model.GetLMSChannelSlug("plg", plgChannelId)
	capstoneChannelId := "capstone_slug"
	capstoneChannelSlug := model.GetLMSChannelSlug("capstone", capstoneChannelId)

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

	plgChannel := &model.Channel{
		DisplayName: "old plg dn",
		Name:        plgChannelSlug,
		Type:        model.CHANNEL_PRIVATE,
		TeamId:      th.BasicTeam.Id,
		CreatorId:   th.BasicUser.Id,
	}

	capstoneChannel := &model.Channel{
		DisplayName: "old capstone dn",
		Name:        capstoneChannelSlug,
		Type:        model.CHANNEL_PRIVATE,
		TeamId:      th.BasicTeam.Id,
		CreatorId:   th.BasicUser.Id,
	}

	var err *model.AppError
	if plgChannel, err = th.App.CreateChannel(plgChannel, true); err != nil {
		t.Errorf("Expected nil, got %s", err)
	}
	if _, err = th.App.CreateChannel(capstoneChannel, true); err != nil {
		t.Errorf("Expected nil, got %s", err)
	}

	err = th.App.SyncLTIChannels(lms, launchData)
	assert.Nil(t, err)

	plgChannel, err = th.App.GetChannelByName(plgChannelSlug, th.BasicTeam.Id, false)
	if err != nil {
		t.Errorf("Expected nil, got %s", err)
	}
	assert.Equal(t, "Plg DN", plgChannel.DisplayName)

	capstoneChannel, err = th.App.GetChannelByName(capstoneChannelSlug, th.BasicTeam.Id, false)
	if err != nil {
		t.Errorf("Expected nil, got %s", err)
	}
	assert.Equal(t, "Capstone DN", capstoneChannel.DisplayName)
}
