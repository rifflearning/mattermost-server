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

	if err := r.ParseForm(); err != nil {
		mlog.Error("Error occurred while parsing submited form: " + err.Error())
		c.Err = model.NewAppError("loginWithLti", "api.lti.login.app_error", nil, "", http.StatusBadRequest)
	}

	// printing launch data for debugging purpose
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

	mlog.Debug("Validating LTI request")
	consumerKey := r.FormValue("oauth_consumer_key")
	lms := c.App.GetLMSToUse(consumerKey)

	if ok := lms.ValidateLTIRequest(c.GetSiteURLHeader()+c.Path, r); !ok {
		c.Err = model.NewAppError("loginWithLti", "api.lti.login.app_error", nil, "", http.StatusNotImplemented)
		return
	}

	setLTIDataCookie(c, w, r)

	mlog.Debug("Redirecting to the LTI signup page")
	http.Redirect(w, r, c.GetSiteURLHeader()+"/signup_lti", http.StatusFound)
}

func encodeLTIRequest(v url.Values) string {
	if v == nil {
		return ""
	}
	form := make(map[string]string)
	for key, value := range v {
		form[key] = value[0]
	}
	res, err := json.Marshal(form)
	if err != nil {
		mlog.Error("Error in json.Marshal: " + err.Error())
		return ""
	}

	return base64.StdEncoding.EncodeToString([]byte(string(res)))
}

func setLTIDataCookie(c *Context, w http.ResponseWriter, r *http.Request) {
	r.ParseForm() // to populate r.Form
	maxAge := 600 // 10 minutes
	expiresAt := time.Unix(model.GetMillis()/1000+int64(maxAge), 0)
	cookie := &http.Cookie{
		Name:     model.LTI_LAUNCH_DATA_COOKIE,
		Value:    encodeLTIRequest(r.Form),
		Path:     "/",
		MaxAge:   maxAge,
		Expires:  expiresAt,
		Domain:   c.App.GetCookieDomain(),
		HttpOnly: false,
	}

	http.SetCookie(w, cookie)
}
