package reCAPTCHAv3

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type reCaptchaResponse struct {
	Success     bool      `json:"success"`
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []int     `json:"error-codes"`
	Score		float32   `json:"score"`
}

func ValidateReCaptcha(SecretKey string, Token string, Domain string) (err error) {
	endpoint := "https://www.google.com/recaptcha/api/siteverify"
	captchaPayload := url.Values{
		"secret": {SecretKey},
		"response": {Token},
	}
	responsePost, err := http.PostForm(endpoint, captchaPayload)
	if err != nil {
		return
	}
	defer responsePost.Body.Close()
	body, err := ioutil.ReadAll(responsePost.Body)
	if err != nil {
		return
	}
	var re reCaptchaResponse
	json.Unmarshal(body, &re)
	if !re.Success || re.Hostname != Domain {
		err = errors.New("Failed reCaptcha")
		return
	}
	return
}