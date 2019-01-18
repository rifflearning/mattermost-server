// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package model

import (
	"encoding/json"

	"github.com/mattermost/mattermost-server/mlog"
)

const (
	LMS_TYPE_FIELD = "Type"

	LMS_TYPE_EDX = "edx"
)

type LMSOAuthSettings struct {
	ConsumerKey    string
	ConsumerSecret string
}

type LMSSettings struct {
	Name  string
	Type  string
	OAuth LMSOAuthSettings
}

type EdxChannel struct {
	CourseType   string
	NameProperty string
	IDProperty   string
}

type EdxUserChannelsSettings struct {
	Type        string
	ChannelList []EdxChannel
}

type EdxLMSSettings struct {
	LMSSettings
	UserChannels EdxUserChannelsSettings
}

type LTISettings struct {
	Enable bool
	LMSs   []interface{}
}

// GetKnownLMSs can be used to extract a slice of known LMSs from LTI settings
func (l *LTISettings) GetKnownLMSs() []interface{} {
	var ret []interface{}
	for _, lms := range l.LMSs {
		enc, err := json.Marshal(lms)
		if err != nil {
			mlog.Error("Error in json.Marshal: " + err.Error())
			continue
		}
		switch lms.(map[string]interface{})[LMS_TYPE_FIELD].(string) {
		case LMS_TYPE_EDX:
			var decodedEdx EdxLMSSettings
			if json.Unmarshal(enc, &decodedEdx) == nil {
				ret = append(ret, decodedEdx)
				continue
			}
		}
	}
	return ret
}
