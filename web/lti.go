// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package web

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
	"fmt"
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
		return
	}

	launchData := make(map[string]string)
	for k, v := range r.Form {
		launchData[k] = v[0]
	}

	// printing launch data for debugging purposes
	mlog.Debug("LTI Launch Data", mlog.String("URL", c.GetSiteURLHeader()+c.Path), mlog.Any("Body", launchData))

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

	LoginLTIUser(c, w, r, lms, launchData)
}

func encodeLTIRequest(launchData map[string]string) (string, *model.AppError) {
	res, err := json.Marshal(launchData)
	if err != nil {
		mlog.Error("Error in json.Marshal: " + err.Error())
		return "", model.NewAppError("encodeLTIRequest", "api.lti.login.marshalling.app_error", nil, "", http.StatusBadRequest)
	}

	return base64.StdEncoding.EncodeToString([]byte(string(res))), nil
}

func setLTIDataCookie(c *Context, w http.ResponseWriter, launchData map[string]string) *model.AppError {
	encodedRequest, appError := encodeLTIRequest(launchData)
	if appError != nil {
		return appError
	}

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
	return nil
}

func LoginLTIUser(c *Context, w http.ResponseWriter, r *http.Request, lms model.LMS, launchData map[string]string) {
	ltiUserID := lms.GetUserId(launchData)
	email := lms.GetEmail(launchData)

	user, err := c.App.GetLTIUser(ltiUserID, email)
	fmt.Println(user, err)
	if customID, ok := user.Props[model.LTI_USER_ID_PROP_KEY]; ok && customID == ltiUserID {
		// user found by lti user id
		// todo: sync user
		// todo: onboard user
		loginUser(c, w, r, user)

		// todo: redirect to channel instead
		http.Redirect(w, r, c.GetSiteURLHeader(), http.StatusFound)
	} else if user.Email == email {
		if customID, ok := user.Props[model.LTI_USER_ID_PROP_KEY]; !ok || customID == "" {
			// if the mm user found by email is not linked to any lms account
			// todo: patch user
			// todo: sync user
			// todo: onboard user
			loginUser(c, w, r, user)

			// todo: redirect to channel instead
			http.Redirect(w, r, c.GetSiteURLHeader(), http.StatusFound)
			return
		} else {
			// if the mm user found by email is linked to a different lms account
			c.Err = model.NewAppError("loginWithLTI", "api.lti.login.cross_linked_users.app_error", nil, "", http.StatusBadRequest)
			return
		}
	} else {
		// if no mm user found by email or lti_user_id
		c.Logout(w, r)
		if err := setLTIDataCookie(c, w, launchData); err != nil {
			c.Err = err
			return
		}
		http.Redirect(w, r, c.GetSiteURLHeader()+"/signup_lti", http.StatusFound)
		return
	}
}

func loginUser(c *Context, w http.ResponseWriter, r *http.Request, user *model.User) {
	session, appErr := c.App.DoLogin(w, r, user, "")
	if appErr != nil {
		c.Err = appErr
		return
	}

	c.Session = *session
}
