package main

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/pkg/errors"
)

type OpenstackClient struct {
	gophercloud.AuthOptions
	Provider *gophercloud.ProviderClient
	Identity *gophercloud.ServiceClient
}

func NewOpenstackClient() *OpenstackClient {
	return &OpenstackClient{
		gophercloud.AuthOptions{
			AllowReauth: true,
		}, nil, nil,
	}
}

func (o *OpenstackClient) Setup() error {
	var err error

	if o.Provider, err = openstack.AuthenticatedClient(o.AuthOptions); err != nil {
		return errors.Wrap(err, "Creating Gophercloud ProviderClient failed")
	}

	if o.Identity, err = openstack.NewIdentityV3(o.Provider, gophercloud.EndpointOpts{}); err != nil {
		return errors.Wrap(err, "Creating Identity ServiceClient failed")
	}

	return nil
}
