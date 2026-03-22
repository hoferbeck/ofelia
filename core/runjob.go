package core

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gobs/args"
)

type RunJob struct {
	BareJob `mapstructure:",squash"`
	Client  *client.Client `json:"-"`
	User    string         `default:"root"`

	TTY bool `default:"false"`

	// do not use bool values with "default:true" because if
	// user would set it to "false" explicitly, it still will be
	// changed to "true" https://github.com/mcuadros/ofelia/issues/135
	// so lets use strings here as workaround
	Delete string `default:"true"`
	Pull   string `default:"true"`

	Image       string
	Network     string
	Hostname    string
	Container   string
	Volume      []string
	VolumesFrom []string `gcfg:"volumes-from" mapstructure:"volumes-from,"`
	Environment []string

	containerID string
}

func NewRunJob(c *client.Client) *RunJob {
	return &RunJob{Client: c}
}

func (j *RunJob) Run(ctx *Context) error {
	var ctr *container.InspectResponse
	var err error
	pull, _ := strconv.ParseBool(j.Pull)

	if j.Image != "" && j.Container == "" {
		if err = func() error {
			var pullError error

			// if Pull option "true"
			// try pulling image first
			if pull {
				if pullError = j.pullImage(); pullError == nil {
					ctx.Log("Pulled image " + j.Image)
					return nil
				}
			}

			// if Pull option "false"
			// try to find image locally first
			searchErr := j.searchLocalImage()
			if searchErr == nil {
				ctx.Log("Found locally image " + j.Image)
				return nil
			}

			// if couldn't find image locally, still try to pull
			if !pull && searchErr == ErrLocalImageNotFound {
				if pullError = j.pullImage(); pullError == nil {
					ctx.Log("Pulled image " + j.Image)
					return nil
				}
			}

			if pullError != nil {
				return pullError
			}

			if searchErr != nil {
				return searchErr
			}

			return nil
		}(); err != nil {
			return err
		}

		ctr, err = j.buildContainer()
		if err != nil {
			return err
		}
	} else {
		containerInspect, inspectErr := j.Client.ContainerInspect(context.Background(), j.Container)
		err = inspectErr
		if err != nil {
			return err
		}
		ctr = &containerInspect
	}

	if ctr != nil {
		j.containerID = ctr.ID
	}

	// cleanup container if it is a created one
	if j.Container == "" {
		defer func() {
			if delErr := j.deleteContainer(); delErr != nil {
				ctx.Warn("failed to delete container: " + delErr.Error())
			}
		}()
	}

	startTime := time.Now()
	if err := j.startContainer(); err != nil {
		return err
	}

	err = j.watchContainer()
	if err == ErrUnexpected {
		return err
	}

	logs, logsErr := j.Client.ContainerLogs(context.Background(), ctr.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      strconv.FormatInt(startTime.Unix(), 10),
	})
	if logsErr != nil {
		ctx.Warn("failed to fetch container logs: " + logsErr.Error())
	} else {
		defer logs.Close()
		if j.TTY {
			_, logsErr = io.Copy(ctx.Execution.OutputStream, logs)
		} else {
			_, logsErr = stdcopy.StdCopy(ctx.Execution.OutputStream, ctx.Execution.ErrorStream, logs)
		}
		if logsErr != nil {
			ctx.Warn("failed to fetch container logs: " + logsErr.Error())
		}
	}

	return err
}

func (j *RunJob) searchLocalImage() error {
	imgs, err := j.Client.ImageList(context.Background(), buildFindLocalImageOptions(j.Image))
	if err != nil {
		return err
	}

	if len(imgs) != 1 {
		return ErrLocalImageNotFound
	}

	return nil
}

func (j *RunJob) pullImage() error {
	ref, opts := buildPullOptions(j.Image)
	stream, err := j.Client.ImagePull(context.Background(), ref, opts)
	if err != nil {
		return fmt.Errorf("error pulling image %q: %s", j.Image, err)
	}
	defer stream.Close()
	_, _ = io.Copy(io.Discard, stream)

	return nil
}

func (j *RunJob) buildContainer() (*container.InspectResponse, error) {
	resp, err := j.Client.ContainerCreate(context.Background(),
		&container.Config{
			Image:        j.Image,
			AttachStdin:  false,
			AttachStdout: true,
			AttachStderr: true,
			Tty:          j.TTY,
			Cmd:          args.GetArgs(j.Command),
			User:         j.User,
			Env:          j.Environment,
			Hostname:     j.Hostname,
		},
		&container.HostConfig{
			Binds:       j.Volume,
			VolumesFrom: j.VolumesFrom,
		},
		&network.NetworkingConfig{},
		nil,
		"",
	)

	if err != nil {
		return nil, fmt.Errorf("error creating exec: %s", err)
	}

	if j.Network != "" {
		networkOpts := network.ListOptions{
			Filters: filters.NewArgs(filters.Arg("name", j.Network)),
		}
		if networks, err := j.Client.NetworkList(context.Background(), networkOpts); err == nil {
			for _, netSummary := range networks {
				if err := j.Client.NetworkConnect(context.Background(), netSummary.ID, resp.ID, &network.EndpointSettings{}); err != nil {
					return nil, fmt.Errorf("error connecting container to network: %s", err)
				}
			}
		}
	}

	c, err := j.Client.ContainerInspect(context.Background(), resp.ID)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (j *RunJob) startContainer() error {
	return j.Client.ContainerStart(context.Background(), j.containerID, container.StartOptions{})
}

func (j *RunJob) stopContainer(timeout uint) error {
	stopTimeout := int(timeout)
	return j.Client.ContainerStop(context.Background(), j.containerID, container.StopOptions{Timeout: &stopTimeout})
}

func (j *RunJob) getContainer() (*container.InspectResponse, error) {
	container, err := j.Client.ContainerInspect(context.Background(), j.containerID)
	if err != nil {
		return nil, err
	}
	return &container, nil
}

const (
	watchDuration      = time.Millisecond * 100
	maxProcessDuration = time.Hour * 24
)

func (j *RunJob) watchContainer() error {
	var s *container.State
	var r time.Duration
	for {
		time.Sleep(watchDuration)
		r += watchDuration

		if r > maxProcessDuration {
			return ErrMaxTimeRunning
		}

		c, err := j.Client.ContainerInspect(context.Background(), j.containerID)
		if err != nil {
			return err
		}

		if !c.State.Running {
			s = c.State
			break
		}
	}

	switch s.ExitCode {
	case 0:
		return nil
	case -1:
		return ErrUnexpected
	default:
		return fmt.Errorf("error non-zero exit code: %d", s.ExitCode)
	}
}

func (j *RunJob) deleteContainer() error {
	if delete, _ := strconv.ParseBool(j.Delete); !delete {
		return nil
	}

	return j.Client.ContainerRemove(context.Background(), j.containerID, container.RemoveOptions{})
}
