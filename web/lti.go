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
)

func (w *Web) InitLti() {
	w.MainRouter.Handle("/login/lti", w.NewHandler(loginWithLTI)).Methods("POST")
}

func loginWithLTI(c *Context, w http.ResponseWriter, r *http.Request) {
	mlog.Debug("Received an LTI Login request")

	mlog.Debug("Testing whether LTI is enabled: " + strconv.FormatBool(c.App.Config().LTISettings.Enable))
	if !c.App.Config().LTISettings.Enable {
		mlog.Error("LTI login request when LTI is disabled in config.json")
		c.Err = model.NewAppError("loginWithLTI", "web.lti.login.disabled.app_error", nil, "", http.StatusNotImplemented)
		return
	}

	// to populate r.Form
	if err := r.ParseForm(); err != nil {
		mlog.Error("Error occurred while parsing submitted form: " + err.Error())
		c.Err = model.NewAppError("loginWithLTI", "web.lti.login.parse.app_error", nil, "", http.StatusBadRequest)
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
		c.Err = model.NewAppError("loginWithLTI", "web.lti.login.no_lms_found", nil, "", http.StatusBadRequest)
		return
	}

	if c.App.Config().LTISettings.EnableSignatureValidation && !lms.ValidateLTIRequest(c.GetSiteURLHeader()+c.Path, r) {
		c.Err = model.NewAppError("loginWithLTI", "web.lti.login.validation.app_error", nil, "", http.StatusBadRequest)
		return
	}

	if err := LoginLTIUser(c, w, r, lms, launchData); err != nil {
		c.Err = err
		return
	}
}

func encodeLTIRequest(launchData map[string]string) (string, *model.AppError) {
	res, err := json.Marshal(launchData)
	if err != nil {
		mlog.Error("Error in json.Marshal: " + err.Error())
		return "", model.NewAppError("encodeLTIRequest", "web.lti.login.marshalling.app_error", nil, "", http.StatusBadRequest)
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

func LoginLTIUser(c *Context, w http.ResponseWriter, r *http.Request, lms model.LMS, launchData map[string]string) *model.AppError {
	ltiUserID := lms.GetUserId(launchData)
	email := lms.GetEmail(launchData)

	user, err := c.App.GetLTIUser(ltiUserID, email)
	if err != nil {
		// Case: User not found
		c.Logout(w, r)
		if err := setLTIDataCookie(c, w, launchData); err != nil {
			return err
		}
		http.Redirect(w, r, c.GetSiteURLHeader()+"/signup_lti", http.StatusFound)
		return nil
	}

	if user.Email == email {
		if customID, ok := user.Props[model.LTI_USER_ID_PROP_KEY]; ok && customID != ltiUserID {
			// Case: MM User linked to different LTI user
			return model.NewAppError("LoginLTIUser", "web.lti.login.cross_linked_users.app_error", nil, "", http.StatusBadRequest)
		} else if !ok || customID == "" {
			// Case: MM User found by email but not linked to any lti user
			user, err = c.App.PatchLTIUser(user.Id, lms, launchData)
			if err != nil {
				return model.NewAppError("LoginLTIUser", "web.lti.login.patch_user.app_error", nil, "", err.StatusCode)
			}

			if err := c.App.OnboardLTIUser(user.Id, lms, launchData); err != nil {
				return model.NewAppError("LoginLTIUser", "web.lti.login.onboard_user.app_error", nil, "", err.StatusCode)
			}
		}
	}

	user, err = c.App.SyncLTIUser(user.Id, lms, launchData)
	if err != nil {
		return model.NewAppError("LoginLTIUser", "web.lti.login.sync_user.app_error", nil, "", err.StatusCode)
	}

	c.Logout(w, r)
	if err := CompleteLTILogin(c, w, r, user); err != nil {
		return err
	}

	return nil
}

func CompleteLTILogin(c *Context, w http.ResponseWriter, r *http.Request, user *model.User) *model.AppError {
	session, err := c.App.DoLogin(w, r, user, "")
	if err != nil {
		return model.NewAppError("CompleteLTILogin", "web.lti.login.login_user.app_error", nil, "", err.StatusCode)
	}

	c.Session = *session

	// todo: redirect to channel instead
	http.Redirect(w, r, c.GetSiteURLHeader(), http.StatusFound)
	return nil
}
