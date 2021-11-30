package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/runtime"
	"log"
	"net/url"
	"path"
	"strings"
)

const (
	GrafanaURL   = "http://localhost:3001/"
	authUsername = "admin"
	authPassword = "prom-operator"
)

type Client struct {
	baseURL     string
	key         string
	isBasicAuth bool
	username    string
	password    string
	client      *resty.Client
}

type Dashboard struct {
	Dashboard *runtime.RawExtension `json:"dashboard,omitempty"`
	FolderId  int                   `json:"folderId,omitempty"`
	FolderUid string                `json:"FolderUid,omitempty"`
	Message   string                `json:"message,omitempty"`
	Overwrite bool                  `json:"overwrite,omitempty"`
}

type GrafanaResponse struct {
	ID      *int    `json:"id,omitempty"`
	UID     *string `json:"uid,omitempty"`
	URL     *string `json:"url,omitempty"`
	Title   *string `json:"title,omitempty"`
	Message *string `json:"message,omitempty"`
	Status  *string `json:"status,omitempty"`
	Version *int    `json:"version,omitempty"`
	Slug    *string `json:"slug,omitempty"`
}

// NewClient initializes client for interacting with an instance of Grafana server;
// apiKeyOrBasicAuth accepts either 'username:password' basic authentication credentials,
// or a Grafana API key. If it is an empty string then no authentication is used.
func NewClient(hostURL string, keyOrBasicAuth string) (*Client, error) {
	isBasicAuth := strings.Contains(keyOrBasicAuth, ":")
	baseURL, err := url.Parse(hostURL)
	if err != nil {
		return nil, err
	}
	client := &Client{
		baseURL:     baseURL.String(),
		key:         "",
		isBasicAuth: isBasicAuth,
		username:    "",
		password:    "",
		client:      resty.New(),
	}
	if len(keyOrBasicAuth) > 0 {
		if !isBasicAuth {
			client.key = keyOrBasicAuth
		} else {
			auths := strings.Split(keyOrBasicAuth, ":")
			if len(auths) != 2 {
				return nil, errors.New("given basic auth format is invalid. expected format: <username>:<password>")
			}
			client.username = auths[0]
			client.password = auths[1]
		}
	}
	return client, nil
}

// SetDashboard will create or update grafana dashboard
func (c *Client) SetDashboard(ctx context.Context, db *Dashboard) (*GrafanaResponse, error) {
	u, _ := url.Parse(c.baseURL)
	u.Path = path.Join(u.Path, "api/dashboards/db")
	var resp *resty.Response
	var err error
	if c.isBasicAuth {
		resp, err = c.client.R().SetContext(ctx).SetBasicAuth(c.username, c.password).SetBody(db).Post(u.String())
	} else {
		resp, err = c.client.R().SetContext(ctx).SetAuthToken(c.key).SetBody(db).Post(u.String())
	}
	if err != nil {
		return nil, err
	}
	gResp := &GrafanaResponse{}
	err = json.Unmarshal(resp.Body(), gResp)
	if err != nil {
		return nil, err
	}
	return gResp, nil
}

func (c *Client) DeleteDashboardByUID(ctx context.Context, uid string) (*GrafanaResponse, error) {
	u, _ := url.Parse(c.baseURL)
	u.Path = path.Join(u.Path, fmt.Sprintf("api/dashboards/uid/%v", uid))
	var resp *resty.Response
	var err error
	if c.isBasicAuth {
		resp, err = c.client.R().SetContext(ctx).SetBasicAuth(c.username, c.password).Delete(u.String())
	} else {
		resp, err = c.client.R().SetContext(ctx).SetAuthToken(c.key).Delete(u.String())
	}
	if err != nil {
		return nil, err
	}
	gResp := &GrafanaResponse{}
	err = json.Unmarshal(resp.Body(), gResp)
	if err != nil {
		return nil, err
	}
	return gResp, nil
}

func main() {
	model, err := ioutil.ReadFile("static/model.json")
	if err != nil {
		panic(err)
	}
	//client := resty.New()
	client, err := NewClient(GrafanaURL, fmt.Sprintf("%s:%s", authUsername, authPassword))
	if err != nil {
		panic(err)
	}

	db := &Dashboard{
		Dashboard: &runtime.RawExtension{Raw: model},
		FolderId:  0,
		Message:   "",
		Overwrite: true,
	}

	// create dashboard
	log.Println("=============== Creating Dashboard ================")
	//resp, err := client.R().SetBasicAuth(authUsername, authPassword).SetBody(db).Post("http://localhost:3001/api/dashboards/db")
	//if err != nil {
	//	panic(err)
	//}
	//dbResp := &GrafanaResponse{}
	//err = json.Unmarshal(resp.Body(), dbResp)
	//if err != nil {
	//	panic(err)
	//}

	resp, err := client.SetDashboard(context.TODO(), db)
	if err != nil {
		panic(err)
	}
	fmt.Println("Dashboard creation status: ", *resp.Status)

	log.Println("\n press any key to continue")
	fmt.Scanln()

	// delete dashboard with uid
	log.Println("===================== Deleting Dashboard ====================")
	//deleteURL := fmt.Sprintf("http://localhost:3001/api/dashboards/uid/%v", dbResp.UID)
	//_, err = client.R().SetBasicAuth(authUsername, authPassword).Delete(deleteURL)
	//if err != nil {
	//	panic(err)
	//}

	resp, err = client.DeleteDashboardByUID(context.TODO(), *resp.UID)
	if err != nil {
		panic(err)
	}
	log.Println("Dashboard is successfully deleted")
}
