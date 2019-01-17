package web

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/utils"
)

const (
	LTI_DATA_COOKIE = "MMLTIDATA"
)

func (w *Web) InitLti() {
	w.MainRouter.Handle("/login/lti", w.NewHandler(loginWithLti)).Methods("POST")
}

func loginWithLti(c *Context, w http.ResponseWriter, r *http.Request) {
	if !c.App.Config().LTISettings.Enable {
		mlog.Error("LTI login request when LTI is disabled in config.json")
		c.Err = model.NewAppError("loginWithLti", "api.lti.login.app_error", nil, "", http.StatusNotImplemented)
		return
	}

	valid := isRequestValid(c, r)
	if !valid {
		c.Err = model.NewAppError("loginWithLti", "api.lti.login.app_error", nil, "", http.StatusNotImplemented)
		return
	}

	setLTIDataCookie(c, w, r)

	http.Redirect(w, r, c.GetSiteURLHeader()+"/signup_lti", http.StatusFound)
}

func isRequestValid(c *Context, r *http.Request) bool {
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
		return false
	}

	p := utils.NewProvider(ltiConsumerSecret, c.GetSiteURLHeader()+c.Path)
	p.ConsumerKey = ltiConsumerKey
	if ok, err := p.IsValid(r); err != nil || ok == false {
		mlog.Error("Invalid LTI request: " + err.Error())
		return false
	}

	return true
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
		Name:     LTI_DATA_COOKIE,
		Value:    encodeLTIRequest(r.Form),
		Path:     "/",
		MaxAge:   maxAge,
		Expires:  expiresAt,
		Domain:   c.App.GetCookieDomain(),
		HttpOnly: false,
	}

	http.SetCookie(w, cookie)
}
