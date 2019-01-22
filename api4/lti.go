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

func (api *API) initLTI() {
	api.BaseRoutes.LTI.Handle("/signup", api.ApiHandler(signupWithLTI)).Methods("POST")
}

func signupWithLTI(c *Context, w http.ResponseWriter, r *http.Request) {
	if !c.App.Config().LTISettings.Enable {
		mlog.Error("LTI signup request when LTI is disabled")
		c.Err = model.NewAppError("signupWithLTI", "api.lti.signup.app_error.lti_disabled", nil, "", http.StatusNotImplemented)
		return
	}

	cookie, err := r.Cookie(model.LTI_LAUNCH_DATA_COOKIE)
	if err != nil {
		mlog.Error("Could't extract LTI auth data cookie: " + err.Error())
		c.Err = model.NewAppError("signupWithLTI", "api.lti.signup.app_error.lti_data_cookie_not_found", nil, "", http.StatusBadRequest)
		return
	}

	data, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		mlog.Error("Error occurred while decoding LTI launch data: " + err.Error())
		c.Err = model.NewAppError("signupWithLTI", "api.lti.signup.app_error.lti_data.decoding_failed", nil, "", http.StatusBadRequest)
		return
	}

	ltiLaunchData := map[string]string{}
	if err := json.Unmarshal(data, &ltiLaunchData); err != nil {
		mlog.Error("Error occurred while unmarshaling LTI launch data: " + err.Error())
		c.Err = model.NewAppError("signupWithLTI", "api.lti.signup.app_error.lti_data.unmarshaling_failed", nil, err.Error(), http.StatusBadRequest)
		return
	}

	// validate launch data
	consumerKey := ltiLaunchData["oauth_consumer_key"]
	lms := c.App.GetLMSToUse(consumerKey)

	if !lms.ValidateLTIRequest(c.GetSiteURLHeader()+c.Path, addLaunchDataToForm(ltiLaunchData, r)) {
		c.Err = model.NewAppError("signupWithLTI", "api.lti.signup.app_error.lti_launch_data.validation_failed", nil, "", http.StatusBadRequest)
		return
	}

	// create user
	props := model.MapFromJson(r.Body)
	user := lms.BuildUser(ltiLaunchData, props["password"])
	user, appErr := c.App.CreateUser(user)

	if appErr != nil {
		mlog.Error("Error occurred while creating LTI user: " + appErr.Error())
		c.Err = model.NewAppError("signupWithLTI", "api.lti.signup.create_user_failed", nil, appErr.Error(), appErr.StatusCode)
		return
	}

	// TODO: create required channels here
	// TODO: add user to required channels here

	w.WriteHeader(http.StatusOK)
}

func addLaunchDataToForm(ltiLaunchData map[string]string, request *http.Request) *http.Request {
	request.Form = url.Values{}

	for k, v := range ltiLaunchData {
		request.Form.Set(k, v)
	}

	return request
}
