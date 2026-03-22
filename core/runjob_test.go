package core

import (
	"context"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/fsouza/go-dockerclient/testing"
	logging "github.com/op/go-logging"
	. "gopkg.in/check.v1"
)

const ImageFixture = "test-image"

type SuiteRunJob struct {
	server *testing.DockerServer
	client *client.Client
}

var _ = Suite(&SuiteRunJob{})

func (s *SuiteRunJob) SetUpTest(c *C) {
	var err error
	s.server, err = testing.NewServer("127.0.0.1:0", nil, nil)
	c.Assert(err, IsNil)

	host := "tcp://" + strings.TrimPrefix(strings.TrimSuffix(s.server.URL(), "/"), "http://")
	s.client, err = client.NewClientWithOpts(client.WithHost(host), client.WithVersion("1.27"))
	c.Assert(err, IsNil)

	s.buildImage(c)
	s.createNetwork(c)
}

func (s *SuiteRunJob) TestRun(c *C) {
	job := &RunJob{Client: s.client}
	job.Image = ImageFixture
	job.Command = `echo -a "foo bar"`
	job.User = "foo"
	job.TTY = true
	job.Delete = "true"
	job.Network = "foo"
	job.Hostname = "test-host"
	job.Name = "test"
	job.Environment = []string{"test_Key1=value1", "test_Key2=value2"}
	job.Volume = []string{"/test/tmp:/test/tmp:ro", "/test/tmp:/test/tmp:rw"}

	ctx := &Context{}
	ctx.Execution = NewExecution()
	logging.SetFormatter(logging.MustStringFormatter(logFormat))
	ctx.Logger = logging.MustGetLogger("ofelia")
	ctx.Job = job

	go func() {
		// Docker Test Server doesn't actually start container
		// so "job.Run" will hang until container is stopped
		if err := job.Run(ctx); err != nil {
			c.Fatal(err)
		}
	}()

	time.Sleep(200 * time.Millisecond)
	ctr, err := job.getContainer()
	c.Assert(err, IsNil)
	c.Assert([]string(ctr.Config.Cmd), DeepEquals, []string{"echo", "-a", "foo bar"})
	c.Assert(ctr.Config.User, Equals, job.User)
	c.Assert(ctr.Config.Image, Equals, job.Image)
	c.Assert(ctr.State.Running, Equals, true)
	c.Assert(ctr.Config.Env, DeepEquals, job.Environment)

	// this doesn't seem to be working with DockerTestServer
	// c.Assert(container.Config.Hostname, Equals, job.Hostname)
	// c.Assert(container.HostConfig.Binds, DeepEquals, job.Volume)

	// stop container, we don't need it anymore
	err = job.stopContainer(0)
	c.Assert(err, IsNil)

	// wait and double check if container was deleted on "stop"
	time.Sleep(watchDuration * 2)
	ctr, _ = job.getContainer()
	c.Assert(ctr, IsNil)

	containers, err := s.client.ContainerList(context.Background(), container.ListOptions{All: true})
	c.Assert(err, IsNil)
	c.Assert(containers, HasLen, 0)
}

func (s *SuiteRunJob) TestBuildPullImageOptionsBareImage(c *C) {
	o, _ := buildPullOptions("foo")
	c.Assert(o, Equals, "foo:latest")
}

func (s *SuiteRunJob) TestBuildPullImageOptionsVersion(c *C) {
	o, _ := buildPullOptions("foo:qux")
	c.Assert(o, Equals, "foo:qux")
}

func (s *SuiteRunJob) TestBuildPullImageOptionsRegistry(c *C) {
	o, _ := buildPullOptions("quay.io/srcd/rest:qux")
	c.Assert(o, Equals, "quay.io/srcd/rest:qux")
}

func (s *SuiteRunJob) buildImage(c *C) {
	err := BuildTestImage(s.client, ImageFixture)
	c.Assert(err, IsNil)
}

func (s *SuiteRunJob) createNetwork(c *C) {
	_, err := s.client.NetworkCreate(context.Background(), "foo", network.CreateOptions{Driver: "bridge"})
	c.Assert(err, IsNil)
}
