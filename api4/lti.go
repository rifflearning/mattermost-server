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
	"github.com/mattermost/mattermost-server/utils"
)

func (api *API) initLTI() {
	api.BaseRoutes.LTI.Handle("/signup", api.ApiHandler(signupWithLTI)).Methods("POST")
}

func signupWithLTI(c *Context, w http.ResponseWriter, r *http.Request) {
	if !c.App.Config().LTISettings.Enable {
		mlog.Error("LTI signup request when LTI is disabled")
		// TODO: add error message in en.json
		// TODO: decide the correct error code here
		c.Err = model.NewAppError("signupWithLTI", "api.lti.signup.app_error", map[string]interface{}{"ErrorCode": "1"}, "", http.StatusNotImplemented)
		return
	}

	cookie, err := r.Cookie("MMLTIAUTHDATA")
	if err != nil {
		mlog.Error("Could't extract LTI auth data cookie: " + err.Error())
		c.Err = model.NewAppError("signupWithLTI", "api.lti.signup.app_error", map[string]interface{}{"ErrorCode": "2"}, "", http.StatusBadRequest)
		return
	}

	data, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		mlog.Error("Error occurred while decoding LTI launch data: " + err.Error())
		c.Err = model.NewAppError("signupWithLTI", "api.lti.signup.app_error", map[string]interface{}{"ErrorCode": "3"}, "", http.StatusBadRequest)
		return
	}

	ltiLaunchData := map[string]string{}
	if err := json.Unmarshal(data, &ltiLaunchData); err != nil {
		mlog.Error("Error occurred while unmarshaling LTI launch data: " + err.Error())
		c.Err = model.NewAppError("signupWithLTI", "api.lti.signup.app_error", map[string]interface{}{"ErrorCode": "4"}, err.Error(), http.StatusBadRequest)
		return
	}

	// validate launch data
	ltiLaunchDataRequest := buildLTIFormRequest(ltiLaunchData)
	if !utils.ValidateLTIRequest(c.GetSiteURLHeader()+c.Path, c.App.Config().LTISettings.GetKnownLMSs(), ltiLaunchDataRequest) {
		c.Err = model.NewAppError("loginWithLti", "api.lti.signup.app_error", map[string]interface{}{"ErrorCode": "4"}, "", http.StatusBadRequest)
		return
	}

	props := model.MapFromJson(r.Body)

	// create user
	user := &model.User{
		Email:     ltiLaunchData["lis_person_contact_email_primary"],
		Username:  ltiLaunchData["lis_person_sourcedid"],
		FirstName: ltiLaunchData["lis_person_name_given"],
		LastName:  ltiLaunchData["lis_person_name_family"],
		Position:  ltiLaunchData["roles"],
		Password:  props["password"],
		Props: model.StringMap{
			"lti_user_id": ltiLaunchData["custom_user_id"],
		},
	}

	if _, appErr := c.App.CreateUser(user); appErr != nil {
		mlog.Error("Error occurred while creating new user: " + appErr.Error())
		c.Err = model.NewAppError("loginWithLti", "api.lti.signup.app_error", map[string]interface{}{"ErrorCode": "5"}, appErr.Error(), appErr.StatusCode)
		return
	}

	// TODO: create required channels here
	// TODO: add user to required channels here

	w.WriteHeader(http.StatusOK)
}

func buildLTIFormRequest(ltiLaunchData map[string]string) *http.Request {
	request := &http.Request{}
	request.Form = url.Values{}

	for k, v := range ltiLaunchData {
		request.Form.Set(k, v)
	}

	return request
}
