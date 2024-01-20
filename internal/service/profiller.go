package service

import "github.com/eldius/docker-runner/internal/docker"

type Profiler struct {
	d *docker.Client
}

func NewProfiler() (*Profiler, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, err
	}

	return &Profiler{
		d: client,
	}, nil
}
