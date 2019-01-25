// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package api4

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
)

func (api *API) InitLTI() {
	api.BaseRoutes.LTI.Handle("/signup", api.ApiHandler(signupWithLTI)).Methods("POST")
}

func signupWithLTI(c *Context, w http.ResponseWriter, r *http.Request) {
	if !c.App.Config().LTISettings.Enable {
		mlog.Error("LTI signup request when LTI is disabled")
		c.Err = model.NewAppError("signupWithLTI", "api.lti.signup.disabled.app_error", nil, "", http.StatusNotImplemented)
		return
	}

	cookie, err := r.Cookie(model.LTI_LAUNCH_DATA_COOKIE)
	if err != nil {
		mlog.Error("Could't extract LTI auth data cookie: " + err.Error())
		c.Err = model.NewAppError("signupWithLTI", "api.lti.signup.cookie_missing.app_error", nil, "", http.StatusBadRequest)
		return
	}

	data, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		mlog.Error("Error occurred while decoding LTI launch data: " + err.Error())
		c.Err = model.NewAppError("signupWithLTI", "api.lti.signup.decoding.app_error", nil, "", http.StatusBadRequest)
		return
	}

	ltiLaunchData := map[string]string{}
	if err := json.Unmarshal(data, &ltiLaunchData); err != nil {
		mlog.Error("Error occurred while unmarshaling LTI launch data: " + err.Error())
		c.Err = model.NewAppError("signupWithLTI", "api.lti.signup.unmarshaling.app_error", nil, err.Error(), http.StatusBadRequest)
		return
	}

	// validate launch data
	consumerKey := ltiLaunchData["oauth_consumer_key"]
	lms := c.App.GetLMSToUse(consumerKey)
	if lms == nil {
		c.Err = model.NewAppError("signupWithLTI", "api.lti.signup.no_lms_found", nil, "", http.StatusBadRequest)
		return
	}

	if c.App.Config().LTISettings.EnableSignatureValidation && !lms.ValidateLTIRequest(c.GetSiteURLHeader()+"/login/lti", addLaunchDataToForm(ltiLaunchData, r)) {
		c.Err = model.NewAppError("signupWithLTI", "api.lti.signup.validation.app_error", nil, "", http.StatusBadRequest)
		return
	}

	// create user
	props := model.MapFromJson(r.Body)
	user := lms.BuildUser(ltiLaunchData, props["password"])
	if _, appErr := c.App.CreateUser(user); appErr != nil {
		mlog.Error("Error occurred while creating LTI user: " + appErr.Error())
		c.Err = model.NewAppError("signupWithLTI", "api.lti.signup.create_user.app_error", map[string]interface{}{"UserError": appErr.Message}, appErr.Error(), appErr.StatusCode)
		return
	}

	if err := c.App.OnboardLMSUser(user.Id, lms, ltiLaunchData); err != nil {
		c.Err = err
		return
	}

	// TODO: create required channels here
	// TODO: add user to required channels here

	ReturnStatusOK(w)
}

func addLaunchDataToForm(ltiLaunchData map[string]string, request *http.Request) *http.Request {
	request.Form = url.Values{}

	for k, v := range ltiLaunchData {
		request.Form.Set(k, v)
	}

	return request
}
