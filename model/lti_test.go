// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package model

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetKnownLMSs(t *testing.T) {
	var lmss []interface{}
	lmss = append(lmss, map[string]interface{}{
		"Name":                "LMS_Name",
		"Type":                "edx",
		"OAuthConsumerKey":    "consumer_key",
		"OAuthConsumerSecret": "consumer_secret",
		"Teams": map[string]string{
			"context_id_1": "team-1",
		},
		"PersonalChannels": EdxPersonalChannels{
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
		"DefaultChannels": map[string]EdxDefaultChannel{},
	})

	ltiSettings := &LTISettings{
		Enable:                    true,
		EnableSignatureValidation: true,
		LMSs:                      lmss,
	}

	knownLMSs := ltiSettings.GetKnownLMSs()

	assert.Equal(t, 1, len(knownLMSs))
	assert.Equal(t, "LMS_Name", knownLMSs[0].GetName())
	assert.Equal(t, "edx", knownLMSs[0].GetType())
}

func TestGetLMSChannelSlug(t *testing.T) {
	personalChannelName := "plg"
	channelId := "really_long_personal_channel_id"
	channelSlug := GetLMSChannelSlug(personalChannelName, channelId)
	assert.Equal(t, "plg-really_long_person", channelSlug)
}

func Test_baseValidateLTIRequest(t *testing.T) {
	consumerSecret := "secret"
	consumerKey := "edx-appsembler-test_5345"
	requestURL := "http://localhost:8065/login/lti"

	request := http.Request{}
	request.Method = "POST"
	request.Form = url.Values{
		"oauth_consumer_key":               []string{"edx-appsembler-test_5345"},
		"oauth_signature_method":           []string{"HMAC-SHA1"},
		"oauth_version":                    []string{"1.0"},
		"context_id":                       []string{"context-id"},
		"context_label":                    []string{"Test"},
		"context_title":                    []string{"Testing Various Integrations"},
		"custom_cohort_name":               []string{"Cohort A"},
		"custom_cohort_id":                 []string{"99"},
		"custom_team_name":                 []string{"Team1"},
		"custom_team_id":                   []string{"99"},
		"custom_user_id":                   []string{"999"},
		"lis_person_contact_email_primary": []string{"foo@bar.com"},
		"lis_person_name_family":           []string{"Bar"},
		"lis_person_name_full":             []string{"Foo Bar"},
		"lis_person_name_given":            []string{"Foo"},
		"lis_person_sourcedid":             []string{"foo"},
		"lti_version":                      []string{"LTI-1p0"},
		"roles":                            []string{"Instructor"},
		"user_id":                          []string{"user-id"},
		"oauth_signature":                  []string{"UjaH+n/SxA4DvbMZPNxpLKKRga4="},
	}

	valid := baseValidateLTIRequest(consumerSecret, consumerKey, requestURL, &request)
	assert.Equal(t, true, valid)
}
