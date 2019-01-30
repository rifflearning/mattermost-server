// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package web

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
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

	launchData, err := getLTILaunchData(c, r)
	if err != nil {
		c.Err = err
		return
	}

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

	ltiUserID := lms.GetUserId(launchData)
	email := lms.GetEmail(launchData)

	user := c.App.GetLTIUser(ltiUserID, email)
	if user == nil {
		// Case: MM or LTI User not found
		mlog.Debug("MM or LTI User not found")
		c.Logout(w, r)
		if err := setLTIDataCookie(c, w, launchData); err != nil {
			c.Err = err
			return
		}

		mlog.Debug("Redirecting to login page")
		http.Redirect(w, r, c.GetSiteURLHeader()+"/signup_lti", http.StatusFound)
		return
	}

	if user.Email == email {
		if customID, ok := user.Props[model.LTI_USER_ID_PROP_KEY]; ok && customID != ltiUserID {
			// Case: MM User linked to different LTI user
			mlog.Debug("MM User linked to different LTI user")
			c.Err = model.NewAppError("LoginLTIUser", "web.lti.login.cross_linked_users.app_error", nil, "", http.StatusBadRequest)
			return
		} else if !ok || customID == "" {
			// Case: MM User found by email but not linked to any lti user
			mlog.Debug("MM User found by email but not linked to any lti user")
			user, err = c.App.PatchLTIUser(user.Id, lms, launchData)
			if err != nil {
				c.Err = model.NewAppError("LoginLTIUser", "web.lti.login.patch_user.app_error", nil, "", err.StatusCode)
				return
			}

			if err := c.App.OnboardLTIUser(user.Id, lms, launchData); err != nil {
				c.Err = model.NewAppError("LoginLTIUser", "web.lti.login.onboard_user.app_error", nil, "", err.StatusCode)
				return
			}
		}
	}

	mlog.Debug("Logging in the user")
	user, err = c.App.SyncLTIUser(user.Id, lms, launchData)
	if err != nil {
		c.Err = model.NewAppError("LoginLTIUser", "web.lti.login.sync_user.app_error", nil, "", err.StatusCode)
		return
	}

	if err := FinishLTILogin(c, w, r, user, lms, launchData); err != nil {
		c.Err = err
		return
	}

	redirectUrl := GetRedirectUrl(lms, launchData, c.GetSiteURLHeader())
	http.Redirect(w, r, redirectUrl, http.StatusFound)
}

func FinishLTILogin(c *Context, w http.ResponseWriter, r *http.Request, user *model.User, lms model.LMS, launchData map[string]string) *model.AppError {
	session, err := c.App.DoLogin(w, r, user, "")
	if err != nil {
		return model.NewAppError("FinishLTILogin", "web.lti.login.login_user.app_error", nil, "", err.StatusCode)
	}

	c.Session = *session
	return nil
}

func GetRedirectUrl(lms model.LMS, launchData map[string]string, siteURL string) string {
	var redirectUrl string
	teamSlug := lms.GetTeam(launchData)
	channelSlug, err := lms.GetChannel(launchData)
	if err != nil {
		mlog.Error("Error occurred searching for channel to redirect to. Continuing to Mattermost homepage. Error: " +err.Error())
	}

	if channelSlug == "" {
		// redirect to Mattermost homepage
		redirectUrl = siteURL
	} else {
		redirectUrl = fmt.Sprintf("%s/%s/channels/%s", siteURL, teamSlug, channelSlug)
	}

	return redirectUrl
}

func getLTILaunchData(c *Context, r *http.Request) (map[string]string, *model.AppError) {
	// to populate r.Form
	if err := r.ParseForm(); err != nil {
		return nil, model.NewAppError("getLTILaunchData", "web.lti.login.parse.app_error", nil, "", http.StatusBadRequest)
	}

	launchData := make(map[string]string)
	for k, v := range r.Form {
		launchData[k] = v[0]
	}

	// printing launch data for debugging purposes
	mlog.Debug("LTI Launch Data", mlog.String("URL", c.GetSiteURLHeader()+c.Path), mlog.Any("Body", launchData))
	return launchData, nil
}

func encodeLTIRequest(launchData map[string]string) (string, *model.AppError) {
	mlog.Debug("Encoding LTI launch data")
	res, err := json.Marshal(launchData)
	if err != nil {
		mlog.Error("Error in json.Marshal: " + err.Error())
		return "", model.NewAppError("encodeLTIRequest", "web.lti.login.marshalling.app_error", nil, "", http.StatusBadRequest)
	}

	return base64.StdEncoding.EncodeToString([]byte(string(res))), nil
}

func setLTIDataCookie(c *Context, w http.ResponseWriter, launchData map[string]string) *model.AppError {
	mlog.Debug("Setting LTI launch data cookie")
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
