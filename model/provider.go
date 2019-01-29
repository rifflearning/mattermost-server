// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package model

// LTI Tools from: https://github.com/jordic/lti
// OAuth Tools from: github.com/daemonl/go_oauth

// provides support for working with LTI
// more info can be checked at:
//
// https://www.imsglobal.org/activity/learning-tools-interoperability
//
// Basically it can sign http requests and also it can
// verify incoming LMI requests when acting as a Provider

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	oAuthVersion = "1.0"
	SigHMAC      = "HMAC-SHA1"
)

// Provider is an app that can consume LTI messages,
// also a provider could be used to construct messages and sign them
//
//  p := lti.NewProvider("secret", "http://url.com")
//  p.Add("param_name", "vale").
//    Add("other_param", "param2")
//
//  sig, err := p.Sign()
//
// will sign the request, and add the needed fields to the
// Provider.values > Can access it through p.Params()
// It also can be used to Verify and handle, incoming LTI requests.
//
//  p.IsValid(request)
//
// A Provider also holds an internal params url.Values, that can
// be accessed via Get, or Add.
type Provider struct {
	Secret      string
	URL         string
	ConsumerKey string
	Method      string
	values      url.Values
	key         []byte
	r           *http.Request
}

// NewProvider is a provider configured with sensible defaults
// as a signer the HMACSigner is used... (seems that is the most used)
func NewProvider(secret, urlSrv string) *Provider {
	key := url.QueryEscape(secret) + "&" + url.QueryEscape("")

	return &Provider{
		Secret: secret,
		Method: "POST",
		values: url.Values{},
		key:    []byte(key),
		URL:    urlSrv,
	}
}

// HasRole checks if an LTI request has a provided role
func (p *Provider) HasRole(role string) bool {
	ro := strings.Split(p.Get("roles"), ",")
	roles := strings.Join(ro, " ") + " "
	if strings.Contains(roles, role+" ") {
		return true
	}
	return false
}

// Get a value from the Params map in provider
func (p *Provider) Get(k string) string {
	return p.values.Get(k)
}

// Params returns the map of values stored on the LTI request
func (p *Provider) Params() url.Values {
	return p.values
}

// SetParams for a provider
func (p *Provider) SetParams(v url.Values) *Provider {
	p.values = v
	return p
}

// Add a new param to an LTI request
func (p *Provider) Add(k, v string) *Provider {
	if p.values == nil {
		p.values = url.Values{}
	}
	p.values.Set(k, v)
	return p
}

// Empty checks if a key is defined (or has something)
func (p *Provider) Empty(key string) bool {
	if p.values == nil {
		p.values = url.Values{}
	}
	return p.values.Get(key) == ""
}

// Sign a request, adding required fields,
// A request, can be drilled on a template, iterating, over p.Params()
func (p *Provider) Sign() (string, error) {
	if p.Empty("oauth_version") {
		p.Add("oauth_version", oAuthVersion)
	}
	if p.Empty("oauth_timestamp") {
		p.Add("oauth_timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	}
	if p.Empty("oauth_nonce") {
		p.Add("oauth_nonce", nonce())
	}
	if p.Empty("oauth_signature_method") {
		p.Add("oauth_signature_method", SigHMAC)
	}
	p.Add("oauth_consumer_key", p.ConsumerKey)

	signature, err := SignLTIRequest(p.values, p.URL, p.Method, p.key)
	if err == nil {
		p.Add("oauth_signature", signature)
	}
	return signature, err
}

// IsValid returns if lti request is valid, currently only checks
// if signature is correct
func (p *Provider) IsValid(r *http.Request) (bool, error) {
	r.ParseForm()
	p.values = r.Form

	ckey := r.Form.Get("oauth_consumer_key")
	if ckey != p.ConsumerKey {
		return false, fmt.Errorf("invalid consumer key provided")
	}

	if r.Form.Get("oauth_signature_method") != SigHMAC {
		return false, fmt.Errorf("wrong signature method %s", r.Form.Get("oauth_signature_method"))
	}
	signature := r.Form.Get("oauth_signature")
	// log.Printf("REQuest URLS %s", r.RequestURI)
	sig, err := SignLTIRequest(r.Form, p.URL, r.Method, p.key)
	if err != nil {
		return false, err
	}
	if sig == signature {
		return true, nil
	}
	return false, fmt.Errorf("invalid signature, %s, expected %s", sig, signature)
}

// SignLTIRequest - sign an lti request using HMAC containing a u, url, a http method,
// and a secret. ts is a tokenSecret field from the oauth spec,
// that in this case must be empty.
func SignLTIRequest(form url.Values, u, method string, key []byte) (string, error) {
	baseString, err := getBaseString(method, u, form)
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha1.New, key)
	mac.Write([]byte(baseString))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

// KV is a simple struct for holding the array equivalent of map[string]string
type KV struct {
	Key string
	Val string
}

// kvSorter sorts key value arrays according to the oauth method (keys then values)
// Not tested properly (Not sure that 'less than' for oauth and golang are the same)
type kvSorter struct {
	kvs []KV
}

func (s *kvSorter) Len() int {
	return len(s.kvs)
}

func (s *kvSorter) Swap(i, j int) {
	s.kvs[i], s.kvs[j] = s.kvs[j], s.kvs[i]
}

func (s *kvSorter) Less(i, j int) bool {
	if s.kvs[i].Key == s.kvs[j].Key {
		return s.kvs[i].Val < s.kvs[j].Val
	}
	return s.kvs[i].Key < s.kvs[j].Key
}

// OauthKvSort sorts key value arrays according to the oauth method (keys then values)
// Not tested properly (Not sure that 'less than' for oauth and golang are the same)
func OauthKvSort(kv []KV) {
	sorter := kvSorter{kv}
	sort.Sort(&sorter)
}

func getBaseString(m, u string, form url.Values) (string, error) {

	var kv []KV
	for k := range form {
		if k != "oauth_signature" {
			s := KV{
				Key: k,
				Val: form.Get(k),
			}
			kv = append(kv, s)
		}
	}

	str, err := GetOAuthBaseString(m, u, kv)
	if err != nil {
		return "", err
	}
	// ugly patch for formatting string as expected.
	str = strings.Replace(str, "%2B", "%2520", -1)
	return str, nil
}

// GetOAuthBaseString returns the 'Signature Base String', which is to be encoded as the signature
func GetOAuthBaseString(method, requestUrl string, allParameters []KV) (string, error) {

	for i, kv := range allParameters {
		allParameters[i].Val = url.QueryEscape(kv.Val)
		allParameters[i].Key = url.QueryEscape(kv.Key)
	}

	OauthKvSort(allParameters)

	strs := make([]string, len(allParameters), len(allParameters))
	for i, kv := range allParameters {
		strs[i] = kv.Key + "=" + kv.Val
	}

	urlPart := url.QueryEscape(strings.ToUpper(method)) + "&" + url.QueryEscape(requestUrl)

	return urlPart + "&" + url.QueryEscape(strings.Join(strs, "&")), nil
}

var nonceCounter uint64

// nonce returns a unique string.
func nonce() string {
	n := atomic.AddUint64(&nonceCounter, 1)
	if n == 1 {
		binary.Read(rand.Reader, binary.BigEndian, &n)
		n ^= uint64(time.Now().UnixNano())
		atomic.CompareAndSwapUint64(&nonceCounter, 1, n)
	}
	return strconv.FormatUint(n, 16)
}
