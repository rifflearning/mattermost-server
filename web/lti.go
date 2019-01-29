// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package web

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
)

func (w *Web) InitLti() {
	w.MainRouter.Handle("/login/lti", w.NewHandler(loginWithLTI)).Methods("POST")
}

func loginWithLTI(c *Context, w http.ResponseWriter, r *http.Request) {
	mlog.Debug("Received an LTI Login request")

	mlog.Debug("Testing whether LTI is enabled: " + strconv.FormatBool(c.App.Config().LTISettings.Enable))
	if !c.App.Config().LTISettings.Enable {
		mlog.Error("LTI login request when LTI is disabled in config.json")
		c.Err = model.NewAppError("loginWithLTI", "api.lti.login.disabled.app_error", nil, "", http.StatusNotImplemented)
		return
	}

	// to populate r.Form
	if err := r.ParseForm(); err != nil {
		mlog.Error("Error occurred while parsing submitted form: " + err.Error())
		c.Err = model.NewAppError("loginWithLTI", "api.lti.login.parse.app_error", nil, "", http.StatusBadRequest)
	}

	// printing launch data for debugging purposes
	body := make(map[string]string)
	for k, v := range r.Form {
		body[k] = v[0]
	}

	mlog.Debug("LTI Launch Data", mlog.String("URL", c.GetSiteURLHeader()+c.Path), mlog.Any("Body", body))

	mlog.Debug("Validate LTI request. LTI Signature Validation enabled: " + strconv.FormatBool(c.App.Config().LTISettings.EnableSignatureValidation))
	consumerKey := r.FormValue("oauth_consumer_key")
	lms := c.App.GetLMSToUse(consumerKey)
	if lms == nil {
		c.Err = model.NewAppError("loginWithLTI", "api.lti.login.no_lms_found", nil, "", http.StatusBadRequest)
		return
	}

	if c.App.Config().LTISettings.EnableSignatureValidation && !lms.ValidateLTIRequest(c.GetSiteURLHeader()+c.Path, r) {
		c.Err = model.NewAppError("loginWithLTI", "api.lti.login.validation.app_error", nil, "", http.StatusBadRequest)
		return
	}

	encodedRequest, appError := encodeLTIRequest(r.Form)
	if appError != nil {
		c.Err = appError
		return
	}
	setLTIDataCookie(c, w, encodedRequest)

	mlog.Debug("Redirecting to the LTI signup page")
	http.Redirect(w, r, c.GetSiteURLHeader()+"/signup_lti", http.StatusFound)
}

func encodeLTIRequest(v url.Values) (string, *model.AppError) {
	if v == nil {
		mlog.Error("The LTI request form data to be encoded is empty.")
		return "", model.NewAppError("encodeLTIRequest", "api.lti.login.encoding.app_error", nil, "", http.StatusBadRequest)
	}

	form := make(map[string]string)
	for key, value := range v {
		form[key] = value[0]
	}

	res, err := json.Marshal(form)
	if err != nil {
		mlog.Error("Error in json.Marshal: " + err.Error())
		return "", model.NewAppError("encodeLTIRequest", "api.lti.login.marshalling.app_error", nil, "", http.StatusBadRequest)
	}

	return base64.StdEncoding.EncodeToString([]byte(string(res))), nil
}

func setLTIDataCookie(c *Context, w http.ResponseWriter, encodedRequest string) {
	maxAge := 600 // 10 minutes
	expiresAt := time.Unix(model.GetMillis()/1000+int64(maxAge), 0)
	cookie := &http.Cookie{
		Name:     model.LTI_LAUNCH_DATA_COOKIE,
		Value:    encodedRequest,
		Path:     "/",
		MaxAge:   maxAge,
		Expires:  expiresAt,
		Domain:   c.App.GetCookieDomain(),
		HttpOnly: false,
	}

	http.SetCookie(w, cookie)
}
