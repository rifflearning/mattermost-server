// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package web

import (
	"fmt"
	"strconv"

	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/utils"
	"net/http"
)

func (w *Web) InitLti() {
	w.MainRouter.Handle("/login/lti", w.NewHandler(loginWithLTI)).Methods("POST")
}

func loginWithLTI(c *Context, w http.ResponseWriter, r *http.Request) {
	mlog.Debug("Received an LTI Login request")

	r.ParseForm()
	mlog.Debug("LTI Launch Data is: ")
	for k, v := range r.Form {
		mlog.Debug(fmt.Sprintf("[%s: %s]", k, v[0]))
	}

	mlog.Debug("Testing whether LTI is enabled: " + strconv.FormatBool(c.App.Config().LTISettings.Enable))
	if !c.App.Config().LTISettings.Enable {
		mlog.Error("LTI login request when LTI is disabled in config.json")
		c.Err = model.NewAppError("loginWithLti", "api.lti.login.error.lti_disabled", nil, "", http.StatusNotImplemented)
		return
	}

	// Validate request
	mlog.Debug("Validating LTI request")
	lmss := c.App.Config().LTISettings.GetKnownLMSs()
	ltiConsumerKey := r.FormValue("oauth_consumer_key")
	var ltiConsumerSecret string

	for _, val := range lmss {
		// TODO: Figure out a better way to find consumer secret for multiple LMSs
		if lms, ok := val.(model.EdxLMSSettings); ok {
			if lms.OAuth.ConsumerKey == ltiConsumerKey {
				ltiConsumerSecret = lms.OAuth.ConsumerSecret
				break
			}
		}
	}

	if ltiConsumerSecret == "" {
		mlog.Error("Consumer secret not found for consumer key: " + ltiConsumerKey)
		c.Err = model.NewAppError("loginWithLti", "api.lti.login.app_error", nil, "", http.StatusNotImplemented)
		return
	}

	p := utils.NewProvider(ltiConsumerSecret, c.GetSiteURLHeader()+c.Path)
	p.ConsumerKey = ltiConsumerKey
	if ok, err := p.IsValid(r); err != nil || ok == false {
		mlog.Error("Invalid LTI request: " + err.Error())
		c.Err = model.NewAppError("loginWithLti", "api.lti.login.app_error", nil, "", http.StatusNotImplemented)
		return
	}

	mlog.Debug("Redirecting to the LTI signup page")
	http.Redirect(w, r, c.GetSiteURLHeader()+"/signup_lti", http.StatusFound)
}
