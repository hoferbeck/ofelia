package core

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/fsouza/go-dockerclient/testing"
	logging "github.com/op/go-logging"

	. "gopkg.in/check.v1"
)

const ServiceImageFixture = "test-image"

type SuiteRunServiceJob struct {
	server *testing.DockerServer
	client *client.Client
}

var _ = Suite(&SuiteRunServiceJob{})

const logFormat = "%{color}%{shortfile} ▶ %{level}%{color:reset} %{message}"

var logger Logger

func (s *SuiteRunServiceJob) SetUpTest(c *C) {
	var err error

	logging.SetFormatter(logging.MustStringFormatter(logFormat))

	logger = logging.MustGetLogger("ofelia")
	s.server, err = testing.NewServer("127.0.0.1:0", nil, nil)
	c.Assert(err, IsNil)

	host := "tcp://" + strings.TrimPrefix(strings.TrimSuffix(s.server.URL(), "/"), "http://")
	s.client, err = client.NewClientWithOpts(client.WithHost(host), client.WithVersion("1.27"))
	c.Assert(err, IsNil)

	s.buildImage(c)
}

func (s *SuiteRunServiceJob) TestRun(c *C) {
	job := &RunServiceJob{Client: s.client}
	job.Image = ServiceImageFixture
	job.Command = `echo -a foo bar`
	job.User = "foo"
	job.TTY = true
	job.Delete = "true"
	job.Network = "foo"

	e := NewExecution()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond * 600)

		tasks, err := s.client.TaskList(context.Background(), swarm.TaskListOptions{})
		if err != nil || len(tasks) == 0 {
			return
		}

		c.Assert(strings.Join(tasks[0].Spec.ContainerSpec.Command, ","), Equals, "echo,-a,foo,bar")

		c.Assert(tasks[0].Status.State, Equals, swarm.TaskStateReady)

		err = s.client.ServiceRemove(context.Background(), tasks[0].ServiceID)

		c.Assert(err, IsNil)
	}()

	err := job.Run(&Context{Execution: e, Logger: logger})
	if err != nil {
		c.Assert(strings.Contains(err.Error(), "404 page not found"), Equals, true)
		wg.Wait()
		return
	}
	wg.Wait()

	containers, err := s.client.TaskList(context.Background(), swarm.TaskListOptions{})

	c.Assert(err, IsNil)
	c.Assert(containers, HasLen, 0)
}

func (s *SuiteRunServiceJob) TestBuildPullImageOptionsBareImage(c *C) {
	o, _ := buildPullOptions("foo")
	c.Assert(o, Equals, "foo:latest")
}

func (s *SuiteRunServiceJob) TestBuildPullImageOptionsVersion(c *C) {
	o, _ := buildPullOptions("foo:qux")
	c.Assert(o, Equals, "foo:qux")
}

func (s *SuiteRunServiceJob) TestBuildPullImageOptionsRegistry(c *C) {
	o, _ := buildPullOptions("quay.io/srcd/rest:qux")
	c.Assert(o, Equals, "quay.io/srcd/rest:qux")
}

func (s *SuiteRunServiceJob) buildImage(c *C) {
	err := BuildTestImage(s.client, ServiceImageFixture)
	c.Assert(err, IsNil)
}
