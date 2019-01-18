package api4

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignupWithLTI(t *testing.T) {
	th := Setup().InitBasic()
	defer th.TearDown()
	Client := th.Client

	launchData := map[string]string{
		"oauth_consumer_key":               "canvas-emeritus_5863",
		"oauth_signature_method":           "HMAC-SHA1",
		"oauth_version":                    "1.0",
		"context_id":                       "context-id",
		"context_label":                    "Test",
		"context_title":                    "Testing Various Integrations",
		"custom_cohort_name":               "Cohort A",
		"custom_cohort_id":                 "99",
		"custom_team_name":                 "Team1",
		"custom_team_id":                   "99",
		"custom_user_id":                   "999",
		"lis_person_contact_email_primary": "foo@bar.com",
		"lis_person_name_family":           "Bar",
		"lis_person_name_full":             "Foo Bar",
		"lis_person_name_given":            "Foo",
		"lis_person_sourcedid":             "foo",
		"lti_version":                      "LTI-1p0",
		"roles":                            "Instructor",
		"user_id":                          "user-id",
		"oauth_signature":                  "A1YyNs3Gdea/6g+Q2MpPkUfQB2I=",
	}
	bytes, err := json.Marshal(launchData)
	assert.Nil(t, err)

	cookie := base64.StdEncoding.EncodeToString(bytes)
	Client.HttpHeader["Cookie"] = "MMLTIAUTHDATA=" + cookie

	resp := Client.SignupLTIUser("pa$$word")

	CheckNoError(t, resp)
}
