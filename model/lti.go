// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package model

import (
	"encoding/json"
	"net/http"

	"github.com/mattermost/mattermost-server/mlog"
)

const (
	LMS_TYPE_FIELD = "Type"

	LMS_TYPE_EDX = "edx"

	LTI_LAUNCH_DATA_COOKIE = "MMLTILAUNCHDATA"

	LTI_USER_ID_PROP_KEY = "lti_user_id"
)

type LMSOAuthSettings struct {
	ConsumerKey    string
	ConsumerSecret string
}

type LMS interface {
	GetName() string
	GetType() string
	GetOAuthConsumerKey() string
	GetOAuthConsumerSecret() string
	GetUserId(launchData map[string]string) string
	ValidateLTIRequest(url string, request *http.Request) bool
	BuildUser(launchData map[string]string, password string) *User
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
