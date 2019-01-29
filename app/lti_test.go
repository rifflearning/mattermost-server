package app

import (
	"github.com/mattermost/mattermost-server/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestApp_OnboardLMSUser(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()

	user := th.BasicUser
	lms := &model.EdxLMS {
		Name: "LMS_Name",
		Type: "edx",
		OAuthConsumerKey: "consumer_key",
		OAuthConsumerSecret: "consumer_secret",
		Teams: map[string]string{
			"context_id_1": th.BasicTeam.Name,
		},
		PersonalChannels: model.EdxPersonalChannels {
			Type: "type",
			ChannelList: map[string]model.EdxChannel {
				"plg": {
					IdProperty: "plg_id_property",
					NameProperty: "plg_name_property",
				},
				"capstone": {
					IdProperty: "capstone_id_property",
					NameProperty: "capstone_name_property",
				},
			},
		},
		DefaultChannels: map[string]model.EdxDefaultChannel {

		},
	}

	launchData := map[string]string{
		"context_id": "context_id_1",
		"plg_id_property": "channel_slug",
		"plg_name_property": "Channel Display Name",
	}

	err := th.App.OnboardLTIUser(user.Id, lms, launchData, true)
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
	launchData := map[string]string {
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
