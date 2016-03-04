package auth

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
)

type token struct {
	Value string `json:"token"`
}

func (tok token) String() string {
	return tok.Value
}

//BearerAuthParams parse Bearer Token on Www-Authenticate header
func BearerAuthParams(r *http.Response) map[string]string {
	s := strings.Fields(r.Header.Get("Www-Authenticate"))

	if len(s) != 2 || s[0] != "Bearer" {
		return nil
	}
	result := map[string]string{}

	for _, kv := range strings.Split(s[1], ",") {
		parts := strings.Split(kv, "=")
		if len(parts) != 2 {
			continue
		}
		result[strings.Trim(parts[0], "\" ")] = strings.Trim(parts[1], "\" ")
	}
	return result
}

func Authenticate(dockerResponse *http.Response, request *http.Request) error {
	bearerToken := BearerAuthParams(dockerResponse)
	url := bearerToken["realm"] + "?service=" + bearerToken["service"] + "&scope=" + bearerToken["scope"]
	res, err := http.Get(url)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return err
	}

	var tok token
	err = json.Unmarshal(body, &tok)

	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+tok.String())

	return nil
}
