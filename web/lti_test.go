// Copyright (c) 2019-present Riff Analytics All Rights Reserved.
// See LICENSE.txt for license information.

package web

import (
	"net/http"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginWithLTI(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()

	pluginJson := map[string]interface{}{
		"enable": false,
		"enablesignaturevalidation": true,
		"lmss": nil,
	}

	th.App.UpdateConfig(func(cfg *model.Config) {
		cfg.PluginSettings.Plugins[model.LTI_PLUGIN_ID] = pluginJson
	})

	LTISettings, ltiErr := th.App.GetLTISettings();
	if(ltiErr != nil){
		require.Nil(t, ltiErr)
		return
	}

	if !LTISettings.Enable {
		resp, err := http.Post(ApiClient.Url+"/login/lti", "", strings.NewReader("123"))
		require.Nil(t, err)
		assert.True(t, resp.StatusCode != http.StatusOK, "should have errored - lti turned off")
		return
	}

	url := ApiClient.Url + "/login/lti"

	t.Run("LTI validate signature", func(t *testing.T) {
		body := "body={\"oauth_consumer_key\":\"canvas-emeritus_5863\",\"oauth_signature_method\":\"HMAC-SHA1\",\"oauth_timestamp\":\"1545331309\",\"oauth_nonce\":\"lk29gzZiuqMka5jUEfWL0JHZFgEtRlMhLmmPsZZz0\",\"oauth_version\":\"1.0\",\"lti_version\":\"LTI-1p0\",\"oauth_callback\":\"about:blank\",\"resource_link_id\":\"eabb2ed57cf5dec85996803535cbccb4c7f62492\",\"oauth_signature\":\"DPh13v4qi2+C4xR+RX1ZBmNmixA=\"}"
		resp, err := http.Post(url, "application/x-www-form-urlencoded", strings.NewReader(body))
		require.Nil(t, err)
		assert.True(t, resp.StatusCode != http.StatusOK, "should have errored - invalid request")
	})
}
