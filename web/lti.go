// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package web

import (
	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/utils"
	"net/http"
)

func (w *Web) InitLti() {
	w.MainRouter.Handle("/login/lti", w.NewHandler(loginWithLTI)).Methods("POST")
}

func loginWithLTI(c *Context, w http.ResponseWriter, r *http.Request) {
	if !c.App.Config().LTISettings.Enable {
		mlog.Error("LTI login request when LTI is disabled in config.json")
		c.Err = model.NewAppError("loginWithLti", "api.lti.login.error.lti_disabled", nil, "", http.StatusNotImplemented)
		return
	}

	// Validate request
	if !utils.ValidateLTIRequest(c.GetSiteURLHeader()+c.Path, c.App.Config().LTISettings.GetKnownLMSs(), r) {
		c.Err = model.NewAppError("loginWithLti", "api.lti.login.app_error", nil, "", http.StatusNotImplemented)
		return
	}

	http.Redirect(w, r, c.GetSiteURLHeader()+"/signup_lti", http.StatusFound)
}
