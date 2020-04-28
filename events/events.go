package events

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

func New(clientID, clientSecret, commitID, buildID string) *Events {
	apiKey, err := exchangeCreds(clientID, clientSecret)
	if err != nil {
		panic(err)
	}

	return &Events{
		apiKey:   apiKey,
		buildID:  buildID,
		commitID: commitID,
		client:   new(http.Client),
	}
}

type Events struct {
	apiKey   string
	buildID  string
	commitID string
	client   *http.Client
}

func (e *Events) Create(dir, evType string, errs ...error) {
	md := map[string]string{
		"service": dir,
		"commit":  e.commitID,
		"build":   e.buildID,
	}
	if len(errs) > 0 {
		md["error"] = errs[0].Error()
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"type": evType, "metadata": md,
	})

	req, _ := http.NewRequest("POST", "https://api.micro.mu/events/create", bytes.NewBuffer(reqBody))
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	rsp, err := e.client.Do(req)
	if err != nil {
		fmt.Println("Unable to connect to the Micro API")
	}
	defer rsp.Body.Close()

	bytes, _ := ioutil.ReadAll(rsp.Body)
	if rsp.StatusCode != http.StatusOK {
		fmt.Printf("Request Error. Status: %v. Response: %v\n", rsp.Status, string(bytes))
	}
}

// exchangeCreds exchanges a client id/secret for a token which
// can be used to call the api
func exchangeCreds(clientID, clientSecret string) (string, error) {
	reqBody, err := json.Marshal(map[string]string{
		"id":     clientID,
		"secret": clientSecret,
	})
	if err != nil {
		return "", errors.New("Invalid Client ID / Secret")
	}

	req, _ := http.NewRequest("POST", "https://api.micro.mu/auth/Login", bytes.NewBuffer(reqBody))
	rsp, err := new(http.Client).Do(req)
	if err != nil {
		return "", errors.New("Error connecting to Micro API")
	}
	defer rsp.Body.Close()

	bytes, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return "", errors.New("Invalid response from Micro API")
	}

	if rsp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Bad Credentials. Status: %v. Response: %v", rsp.Status, string(bytes))
	}

	var data struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(bytes, &data); err != nil {
		return "", errors.New("Invalid response from Micro API")
	}

	return data.Token, nil
}
