// Copyright 2024 The Contaiman Author
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"github.com/containers/podman/v2/libpod"
	"github.com/containers/podman/v2/pkg/bindings"
	"github.com/containers/podman/v2/pkg/bindings/containers"
	"github.com/containers/podman/v2/pkg/bindings/images"
	"github.com/containers/podman/v2/pkg/bindings/pods"
	"github.com/containers/podman/v2/pkg/domain/entities"
	createconfig "github.com/containers/podman/v2/pkg/spec"
	"github.com/containers/podman/v2/pkg/specgen"
)

type Client struct {
	Ctx context.Context

	Host string
}

func NewClient(uri string) (Client, error) {
	ctx, err := bindings.NewConnection(context.Background(), uri)
	if err != nil {
		return Client{}, err
	}

	return Client{
		ctx,
		uri,
	}, nil
}

func (c *Client) PullImage(uri string, options entities.ImagePullOptions) ([]string, error) {
	return images.Pull(c.Ctx, uri, options)
}

func (c *Client) QueryImages() ([]*entities.ImageSummary, error) {
	return images.List(c.Ctx, nil, nil)
}

func (c *Client) QueryContainers() ([]entities.ListContainer, error) {
	return containers.List(c.Ctx, nil, nil, nil, nil, nil, nil)
}

func (c *Client) QueryPods() ([]*entities.ListPodsReport, error) {
	return pods.List(c.Ctx, nil)
}

func (c *Client) CreatePod(name string, config *specgen.PodSpecGenerator) (*entities.PodCreateReport, error) {
	return pods.CreatePodFromSpec(c.Ctx, config)
}

func (c *Client) CreateContainer(name string, config *createconfig.CreateConfig) (*libpod.Container, error) {
	return createconfig.CreateContainerFromCreateConfig(c.Ctx, nil, config, nil)
}
