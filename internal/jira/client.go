package jira

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const (
	restNotifications     = "/gateway/api/notification-log/api/2/notifications"
	restNotificationCount = "/gateway/api/notification-log/api/2/notifications/count/unseen"
	restAuthUrl           = "https://id.atlassian.com/id/rest/login"

	errorUnableToFoundHost = "unable to establish connection to the server. Check URL spelling"
	errorWrongCredentials  = "wrong username or password"
)

// Data
type (
	// notifications is the multiple of notification data object
	Notifications struct {
		Notifications []Notification `json:"data"`
	}

	// notification is a single data object received by API endpoint
	Notification struct {
		Title     string            `json:"title"`
		Users     map[string]string `json:"users"`
		Template  string            `json:"template"`
		Timestamp string            `json:"timestamp"`
		Metadata  Metadata          `json:"metadata"`
	}

	Metadata struct {
		User User `json:"user"`
	}

	User struct {
		AtlassianId string `json:"atlassianId"`
		Name        string `json:"name"`
	}

	Count struct {
		Count int `json:"count"`
	}

	// client is service for our data fetching
	client struct {
		host             string
		client           httpClient
		cookie           string
		credentialBuffer *bytes.Buffer
	}
)

// public types
type (
	Client interface {
		FetchNotificationCount() (int, error)
		FetchNotifications() ([]Notification, error)
		Login() error
	}

	httpClient interface {
		Do(*http.Request) (*http.Response, error)
	}
)

// Factory to our fetcher
func NewClient(host, username, password string) (Client, error) {
	if strings.HasSuffix(host, "/") {
		strings.TrimSuffix(host, "/")
	}

	credentials := make(map[string]string)
	credentials["username"] = username
	credentials["password"] = password

	j, err := json.Marshal(credentials)
	if err != nil {
		return nil, err
	}

	credentialBuffer := bytes.NewBuffer(j)

	return &client{
		host,
		&http.Client{
			Timeout: time.Second * 5,
		},
		"",
		credentialBuffer,
	}, nil
}

func (c *client) isHostExisting() error {
	req, err := http.NewRequest(http.MethodGet, c.host, nil)
	if err != nil {
		return err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}

	switch res.StatusCode {
	case http.StatusNotFound:
		return errors.New(errorUnableToFoundHost)
	}

	return nil
}

func (c *client) isLoggedIn() bool {
	req, err := http.NewRequest(http.MethodGet, c.host+restNotificationCount, nil)
	if err != nil {
		return false
	}

	req.Header.Set("Cookie", c.cookie)
	req.Header.Add("Connection", "keep-alive")
	res, err := c.client.Do(req)
	if err != nil {
		return false
	}

	if res.StatusCode != http.StatusForbidden {
		return true
	}

	return false
}

func (c *client) Login() error {
	if err := c.isHostExisting(); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", restAuthUrl, c.credentialBuffer)
	req.Header.Add("Content-Type", "application/json")
	if err != nil {
		return err
	}

	res, err := c.client.Do(req)

	switch res.StatusCode {
	case http.StatusNotFound:
		return errors.New(errorUnableToFoundHost)
	case http.StatusForbidden:
		return errors.New(errorWrongCredentials)
	}

	c.cookie = res.Header.Get("Set-Cookie")

	return err
}

func (c client) FetchNotificationCount() (int, error) {
	if !c.isLoggedIn() {
		c.Login()
	}

	req, err := http.NewRequest(http.MethodGet, c.host+restNotificationCount, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Cookie", c.cookie)
	req.Header.Add("Connection", "keep-alive")
	res, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}

	var data *Count
	err = json.Unmarshal(body, &data)
	if err != nil {
		return 0, err
	}

	return data.Count, nil
}

//
func (c client) FetchNotifications() ([]Notification, error) {
	if !c.isLoggedIn() {
		c.Login()
	}

	req, err := http.NewRequest(http.MethodGet, c.host+restNotifications, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Cookie", c.cookie)
	req.Header.Add("Connection", "keep-alive")
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var data Notifications
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	return data.Notifications, nil
}
