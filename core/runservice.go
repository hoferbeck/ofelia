package core

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
)

// Note: The ServiceJob is loosely inspired by https://github.com/alexellis/jaas/

type RunServiceJob struct {
	BareJob `mapstructure:",squash"`
	Client  *client.Client `json:"-"`
	User    string         `default:"root"`
	TTY     bool           `default:"false"`
	// do not use bool values with "default:true" because if
	// user would set it to "false" explicitly, it still will be
	// changed to "true" https://github.com/mcuadros/ofelia/issues/135
	// so lets use strings here as workaround
	Delete  string `default:"true"`
	Image   string
	Network string
}

func NewRunServiceJob(c *client.Client) *RunServiceJob {
	return &RunServiceJob{Client: c}
}

func (j *RunServiceJob) Run(ctx *Context) error {
	if err := j.pullImage(); err != nil {
		return err
	}

	svcID, err := j.buildService()

	if err != nil {
		return err
	}

	ctx.Logger.Noticef("Created service %s for job %s\n", svcID, j.Name)

	if err := j.watchContainer(ctx, svcID); err != nil {
		if isNotFoundError(err) {
			return nil
		}
		return err
	}

	return j.deleteService(ctx, svcID)
}

func (j *RunServiceJob) pullImage() error {
	ref, opts := buildPullOptions(j.Image)
	stream, err := j.Client.ImagePull(context.Background(), ref, opts)
	if err != nil {
		if isNotFoundError(err) {
			images, listErr := j.Client.ImageList(context.Background(), buildFindLocalImageOptions(j.Image))
			if listErr == nil && len(images) == 1 {
				return nil
			}
		}
		return fmt.Errorf("error pulling image %q: %s", j.Image, err)
	}
	defer stream.Close()
	_, _ = io.Copy(io.Discard, stream)

	return nil
}

func (j *RunServiceJob) buildService() (string, error) {

	//createOptions := types.ServiceCreateOptions{}

	max := uint64(1)
	var spec swarm.ServiceSpec

	spec.TaskTemplate.ContainerSpec =
		&swarm.ContainerSpec{
			Image: j.Image,
		}

	// Make the service run once and not restart
	spec.TaskTemplate.RestartPolicy =
		&swarm.RestartPolicy{
			MaxAttempts: &max,
			Condition:   swarm.RestartPolicyConditionNone,
		}

	// For a service to interact with other services in a stack,
	// we need to attach it to the same network
	if j.Network != "" {
		spec.Networks = []swarm.NetworkAttachmentConfig{
			swarm.NetworkAttachmentConfig{
				Target: j.Network,
			},
		}
	}

	if j.Command != "" {
		spec.TaskTemplate.ContainerSpec.Command = strings.Split(j.Command, " ")
	}

	resp, err := j.Client.ServiceCreate(context.Background(), spec, swarm.ServiceCreateOptions{})
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

const (

	// TODO are these const defined somewhere in the docker API?
	swarmError = -999
)

var svcChecker = time.NewTicker(watchDuration)

func (j *RunServiceJob) watchContainer(ctx *Context, svcID string) error {
	exitCode := swarmError

	ctx.Logger.Noticef("Checking for service ID %s (%s) termination\n", svcID, j.Name)

	// On every tick, check if all the services have completed, or have error out
	var wg sync.WaitGroup
	wg.Add(1)
	var err error
	started := time.Now()

	go func() {
		defer wg.Done()
		for range svcChecker.C {

			if time.Since(started) > maxProcessDuration {
				err = ErrMaxTimeRunning
				return
			}

			taskExitCode, found := j.findtaskstatus(ctx, svcID)

			if found {
				exitCode = taskExitCode
				return
			}
		}
	}()

	wg.Wait()

	ctx.Logger.Noticef("Service ID %s (%s) has completed with exit code %d\n", svcID, j.Name, exitCode)
	return err
}

func (j *RunServiceJob) findtaskstatus(ctx *Context, taskID string) (int, bool) {
	tasks, err := j.Client.TaskList(context.Background(), swarm.TaskListOptions{
		Filters: filters.NewArgs(filters.Arg("service", taskID)),
	})

	if err != nil {
		ctx.Logger.Errorf("Failed to find task ID %s. Considering the task terminated: %s\n", taskID, err.Error())
		return 0, false
	}

	if len(tasks) == 0 {
		// That task is gone now (maybe someone else removed it. Our work here is done
		return 0, true
	}

	exitCode := 1
	var done bool
	stopStates := []swarm.TaskState{
		swarm.TaskStateComplete,
		swarm.TaskStateFailed,
		swarm.TaskStateRejected,
	}

	for _, task := range tasks {

		stop := false
		for _, stopState := range stopStates {
			if task.Status.State == stopState {
				stop = true
				break
			}
		}

		if stop {

			exitCode = task.Status.ContainerStatus.ExitCode

			if exitCode == 0 && task.Status.State == swarm.TaskStateRejected {
				exitCode = 255 // force non-zero exit for task rejected
			}
			done = true
			break
		}
	}
	return exitCode, done
}

func (j *RunServiceJob) deleteService(ctx *Context, svcID string) error {
	if delete, _ := strconv.ParseBool(j.Delete); !delete {
		return nil
	}

	err := j.Client.ServiceRemove(context.Background(), svcID)

	if isNotFoundError(err) {
		ctx.Logger.Warningf("Service %s cannot be removed. An error may have happened, "+
			"or it might have been removed by another process", svcID)
		return nil
	}

	return err

}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	if errdefs.IsNotFound(err) || cerrdefs.IsNotFound(err) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "404") || strings.Contains(msg, "not found")
}
