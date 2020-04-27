// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package model

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-server/mlog"
)

const (
	LMS_TYPE_FIELD = "Type"

	LMS_TYPE_EDX = "edx"

	LTI_LAUNCH_DATA_COOKIE = "MMLTILAUNCHDATA"

	LTI_NAME_COOKIE = "MMLTINAME"

	LTI_USER_ID_PROP_KEY = "lti_user_id"
)

type LMSOAuthSettings struct {
	ConsumerKey    string
	ConsumerSecret string
}

type LMS interface {
	GetEmail(launchData map[string]string) string
	GetName() string
	GetType() string
	GetOAuthConsumerKey() string
	GetOAuthConsumerSecret() string
	GetPersonalChannelNames() []string
	IsLMSForTeam(teamSlug string) bool
	GetUserId(launchData map[string]string) string
	ValidateLTIRequest(url string, request *http.Request) bool
	BuildUser(launchData map[string]string, password string) (*User, *AppError)
	GetTeam(launchData map[string]string) string
	GetPublicChannelsToJoin(launchData map[string]string) map[string]string
	GetPrivateChannelsToJoin(launchData map[string]string) map[string]string
	GetChannel(launchData map[string]string) (string, *AppError)
	SyncUser(user *User, launchData map[string]string) *User
}

type LTISettings struct {
	Enable                    bool
	EnableSignatureValidation bool
	LMSs                      []interface{}
}

// GetKnownLMSs can be used to extract a slice of known LMSs from LTI settings
func (l *LTISettings) GetKnownLMSs() []LMS {
	var ret []LMS

	for _, lms := range l.LMSs {
		bytes, err := json.Marshal(lms)
		if err != nil {
			mlog.Error("Error in json.Marshal: " + err.Error())
			continue
		}

		switch lms.(map[string]interface{})[LMS_TYPE_FIELD].(string) {
		case LMS_TYPE_EDX:
			var decodedEdx EdxLMS
			if json.Unmarshal(bytes, &decodedEdx) == nil {
				ret = append(ret, &decodedEdx)
			}
		}
	}
	return ret
}

func baseValidateLTIRequest(consumerSecret, consumerKey, url string, request *http.Request) bool {
	p := NewProvider(consumerSecret, url)
	p.ConsumerKey = consumerKey

	if ok, err := p.IsValid(request); err != nil || ok == false {
		mlog.Error("Invalid LTI request: " + err.Error())
		return false
	}

	return true
}

func transformLTIUsername(ltiUsername string) string {
	mattermostUsername := ""

	for _, c := range ltiUsername {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '.' || c == '_' {
			mattermostUsername += string(c)
		}
	}

	return mattermostUsername
}

func GetLMSChannelSlug(personalChannelName, channelId string) string {
	channelSlugRaw := fmt.Sprintf("%s-%s", personalChannelName, channelId)
	// This trim is a patch because creating the channel fails if the slug ends w/ a '-'
	// What we should do is remove all non alphanumeric characters from the channelId and lowercase it
	// and then concatenate and truncate.
	channelSlug := strings.Trim(truncateLMSChannelSlug(channelSlugRaw), "-")
	return channelSlug
}

func truncateLMSChannelSlug(channelSlug string) string {
	end := CHANNEL_NAME_UI_MAX_LENGTH
	if len(channelSlug) < end {
		end = len(channelSlug)
	}

	return channelSlug[0:end]
}
