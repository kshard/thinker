//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package command

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"os/user"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/jdxcode/netrc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type AuthConfig struct {
	// Endpoint URL of MCP server
	Endpoint string

	// Resource Config file (default is ~/.iqrc) to read credentials from.
	RC string

	// Custom HTTP client (if nil, default client will be used)
	Client *http.Client
}

// Configures MCP transport with Bearer token authentication.
func NewAuthTransport(spec AuthConfig) (*mcp.StreamableClientTransport, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}

	file := ".iqrc"
	if spec.RC != "" {
		file = spec.RC
	}

	n, err := netrc.Parse(filepath.Join(usr.HomeDir, file))
	if err != nil {
		return nil, err
	}

	uri, err := url.Parse(spec.Endpoint)
	if err != nil {
		return nil, err
	}

	machine := n.Machine(uri.Host)
	if machine == nil {
		return &mcp.StreamableClientTransport{Endpoint: spec.Endpoint}, nil
	}

	switch machine.Get("provider") {
	case "aws":
		return awsTransport(spec.Endpoint, spec.Client, machine)
	default:
		return defTransport(spec.Endpoint, spec.Client, machine)
	}
}

//------------------------------------------------------------------------------

func defTransport(url string, client *http.Client, machine *netrc.Machine) (*mcp.StreamableClientTransport, error) {
	secret := machine.Get("token")
	if secret == "" {
		return &mcp.StreamableClientTransport{Endpoint: url}, nil
	}

	bearer := machine.Get("type")
	if bearer == "" {
		bearer = "Bearer"
	}

	sock := &transport{
		token:  bearer + " " + secret,
		socket: http.DefaultTransport,
	}

	if client != nil && client.Transport != nil {
		sock.socket = client.Transport
	}

	if client == nil {
		client = &http.Client{}
	}
	client.Transport = sock

	override := machine.Get("url")
	if override != "" {
		url = override
	}

	return &mcp.StreamableClientTransport{
		Endpoint:   url,
		HTTPClient: client,
	}, nil
}

type transport struct {
	token  string
	socket http.RoundTripper
}

func (api *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", api.token)
	return api.socket.RoundTrip(req)
}

//------------------------------------------------------------------------------

func awsTransport(url string, client *http.Client, machine *netrc.Machine) (*mcp.StreamableClientTransport, error) {
	conf, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}

	role := machine.Get("role")
	if role != "" {
		extid := machine.Get("externalId")
		assumed, err := config.LoadDefaultConfig(context.Background(),
			config.WithCredentialsProvider(
				aws.NewCredentialsCache(
					stscreds.NewAssumeRoleProvider(sts.NewFromConfig(conf), role,
						func(aro *stscreds.AssumeRoleOptions) {
							if extid != "" {
								aro.ExternalID = aws.String(extid)
							}
						},
					),
				),
			),
		)
		if err != nil {
			return nil, err
		}
		conf = assumed
	}

	sock := &awsiam{
		config: conf,
		signer: v4.NewSigner(),
		socket: http.DefaultTransport,
	}

	if client != nil && client.Transport != nil {
		sock.socket = client.Transport
	}

	if client == nil {
		client = &http.Client{}
	}
	client.Transport = sock

	return &mcp.StreamableClientTransport{
		Endpoint:   url,
		HTTPClient: client,
	}, nil
}

type awsiam struct {
	config aws.Config
	signer *v4.Signer
	socket http.RoundTripper
}

func (api *awsiam) RoundTrip(req *http.Request) (*http.Response, error) {
	credential, err := api.config.Credentials.Retrieve(req.Context())
	if err != nil {
		return nil, err
	}

	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if req.Body != nil {
		buf := &bytes.Buffer{}
		hasher := sha256.New()
		stream := io.TeeReader(req.Body, hasher)
		if _, err := io.Copy(buf, stream); err != nil {
			return nil, err
		}
		hash = hex.EncodeToString(hasher.Sum(nil))

		req.Body.Close()
		req.Body = io.NopCloser(buf)
	}

	err = api.signer.SignHTTP(
		req.Context(),
		credential,
		req,
		hash,
		"execute-api",
		api.config.Region,
		time.Now(),
	)
	if err != nil {
		return nil, err
	}

	return api.socket.RoundTrip(req)
}
