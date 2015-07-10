package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gogo/protobuf/proto"
	"golang.org/x/net/context"

	"github.com/mesos/mesos-go/auth"
	"github.com/mesos/mesos-go/auth/sasl"
	"github.com/mesos/mesos-go/auth/sasl/mech"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
)

// The docker images that we launch
const (
	MinerServerDockerImage = "derekchiang/p2pool"
	MinerDaemonDockerImage = "derekchiang/cpuminer"
)

// Resource usage of the tasks
const (
	MemPerDaemonTask = 128 // mining shouldn't be memory-intensive
	MemPerServerTask = 256 // I'm just guessing
	CPUPerServerTask = 1   // a miner server does not use much CPU
)

// flags
var (
	master       = flag.String("master", "127.0.0.1:5050", "Master address <ip:port>")
	bitcoindAddr = flag.String("bitcoindAddress", "127.0.0.1", "Address where bitcoind runs")

	// auth
	authProvider = flag.String("mesos_authentication_provider", sasl.ProviderName,
		fmt.Sprintf("Authentication provider to use, default is SASL that supports mechanisms: %+v", mech.ListSupported()))
	mesosAuthPrincipal  = flag.String("mesos_authentication_principal", "", "Mesos authentication principal.")
	mesosAuthSecretFile = flag.String("mesos_authentication_secret_file", "", "Mesos authentication secret file.")
)

// minerScheduler implements the Scheduler interface and stores the state
// needed to scheduler tasks.
type minerScheduler struct {
	// bitcoind RPC credentials
	rpcUser string
	rpcPass string

	// mutable state
	minerServerRunning  bool
	minerServerHostname string
	minerServerPort     int // the port that miner daemons connect to
	//unique task ids
	tasksLaunched        int
	currentDaemonTaskIDs []*mesos.TaskID
}

func newMinerScheduler(user, pass string) *minerScheduler {
	return &minerScheduler{
		rpcUser:              user,
		rpcPass:              pass,
		minerServerRunning:   false,
		tasksLaunched:        0,
		currentDaemonTaskIDs: make([]*mesos.TaskID, 0),
	}
}

func (s *minerScheduler) Registered(driver sched.SchedulerDriver, frameworkID *mesos.FrameworkID, masterInfo *mesos.MasterInfo) {
	log.Infoln("Framework registered with Master ", masterInfo)
}

func (s *minerScheduler) Reregistered(driver sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {
	log.Infoln("Framework Re-Registered with Master ", masterInfo)
}

func (s *minerScheduler) Disconnected(sched.SchedulerDriver) {
	log.Infoln("Framework disconnected with Master")
}

func (s *minerScheduler) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	for _, offer := range offers {
		memResources := util.FilterResources(offer.Resources, func(res *mesos.Resource) bool {
			return res.GetName() == "mem"
		})
		mems := 0.0
		for _, res := range memResources {
			mems += res.GetScalar().GetValue()
		}

		cpuResources := util.FilterResources(offer.Resources, func(res *mesos.Resource) bool {
			return res.GetName() == "cpus"
		})
		cpus := 0.0
		for _, res := range cpuResources {
			cpus += res.GetScalar().GetValue()
		}

		portsResources := util.FilterResources(offer.Resources, func(res *mesos.Resource) bool {
			return res.GetName() == "ports"
		})
		var ports uint64
		for _, res := range portsResources {
			portRanges := res.GetRanges().GetRange()
			for _, portRange := range portRanges {
				ports += portRange.GetEnd() - portRange.GetBegin()
			}
		}

		// If a miner server is running, we start a new miner daemon.  Otherwise, we start a new miner server.
		var tasks []*mesos.TaskInfo
		if !s.minerServerRunning && mems >= MemPerServerTask && cpus >= CPUPerServerTask && ports >= 2 {
			var taskID *mesos.TaskID
			var task *mesos.TaskInfo

			// we need two ports
			var p2poolPort uint64
			var workerPort uint64
			// A rather stupid algorithm for picking two ports
			// The difficulty here is that a range might only include one port,
			// in which case we will need to pick another port from another range.
			for _, res := range portsResources {
				r := res.GetRanges().GetRange()[0]
				begin := r.GetBegin()
				end := r.GetEnd()
				if p2poolPort == 0 {
					p2poolPort = begin
					if workerPort == 0 && (begin+1) <= end {
						workerPort = begin + 1
						break
					}
					continue
				}
				if workerPort == 0 {
					workerPort = begin
					break
				}
			}
			s.tasksLaunched++
			taskID = &mesos.TaskID{
				Value: proto.String("miner-server-" + strconv.Itoa(s.tasksLaunched)),
			}

			containerType := mesos.ContainerInfo_DOCKER
			task = &mesos.TaskInfo{
				Name:    proto.String("task-" + taskID.GetValue()),
				TaskId:  taskID,
				SlaveId: offer.SlaveId,
				Container: &mesos.ContainerInfo{
					Type: &containerType,
					Docker: &mesos.ContainerInfo_DockerInfo{
						Image: proto.String(MinerServerDockerImage),
					},
				},
				Command: &mesos.CommandInfo{
					Shell: proto.Bool(false),
					Arguments: []string{
						// these arguments will be passed to run_p2pool.py
						"--bitcoind-address", *bitcoindAddr,
						"--p2pool-port", strconv.Itoa(int(p2poolPort)),
						"-w", strconv.Itoa(int(workerPort)),
						s.rpcUser, s.rpcPass,
					},
				},
				Resources: []*mesos.Resource{
					util.NewScalarResource("cpus", CPUPerServerTask),
					util.NewScalarResource("mem", MemPerServerTask),
				},
			}
			log.Infof("Prepared task: %s with offer %s for launch\n", task.GetName(), offer.Id.GetValue())

			cpus -= CPUPerServerTask
			mems -= MemPerServerTask

			// update state
			s.minerServerHostname = offer.GetHostname()
			s.minerServerRunning = true
			s.minerServerPort = int(workerPort)

			tasks = append(tasks, task)
		}

		if s.minerServerRunning && mems >= MemPerDaemonTask {
			var taskID *mesos.TaskID
			var task *mesos.TaskInfo

			s.tasksLaunched++
			taskID = &mesos.TaskID{
				Value: proto.String("miner-daemon-" + strconv.Itoa(s.tasksLaunched)),
			}

			containerType := mesos.ContainerInfo_DOCKER
			task = &mesos.TaskInfo{
				Name:    proto.String("task-" + taskID.GetValue()),
				TaskId:  taskID,
				SlaveId: offer.SlaveId,
				Container: &mesos.ContainerInfo{
					Type: &containerType,
					Docker: &mesos.ContainerInfo_DockerInfo{
						Image: proto.String(MinerDaemonDockerImage),
					},
				},
				Command: &mesos.CommandInfo{
					Shell:     proto.Bool(false),
					Arguments: []string{"-o", s.minerServerHostname + ":" + strconv.Itoa(s.minerServerPort)},
				},
				Resources: []*mesos.Resource{
					util.NewScalarResource("cpus", cpus),
					util.NewScalarResource("mem", MemPerDaemonTask),
				},
			}
			log.Infof("Prepared task: %s with offer %s for launch\n", task.GetName(), offer.Id.GetValue())

			tasks = append(tasks, task)
			s.currentDaemonTaskIDs = append(s.currentDaemonTaskIDs, taskID)
		}

		driver.LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, &mesos.Filters{RefuseSeconds: proto.Float64(1)})
	}
}

func (s *minerScheduler) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	log.Infoln("Status update: task", status.TaskId.GetValue(), " is in state ", status.State.Enum().String())
	// If the mining server failed for any reason, kill all daemons, since they will be trying to talk to the failed mining server
	if strings.Contains(status.GetTaskId().GetValue(), "server") &&
		(status.GetState() == mesos.TaskState_TASK_LOST ||
			status.GetState() == mesos.TaskState_TASK_KILLED ||
			status.GetState() == mesos.TaskState_TASK_FINISHED ||
			status.GetState() == mesos.TaskState_TASK_ERROR ||
			status.GetState() == mesos.TaskState_TASK_FAILED) {

		s.minerServerRunning = false

		// kill all tasks
		for _, taskID := range s.currentDaemonTaskIDs {
			_, err := driver.KillTask(taskID)
			if err != nil {
				log.Errorf("Failed to kill task %s", taskID)
			}
		}
		s.currentDaemonTaskIDs = make([]*mesos.TaskID, 0)
	}
}

func (s *minerScheduler) OfferRescinded(sched.SchedulerDriver, *mesos.OfferID) {}
func (s *minerScheduler) FrameworkMessage(sched.SchedulerDriver, *mesos.ExecutorID, *mesos.SlaveID, string) {
}
func (s *minerScheduler) SlaveLost(sched.SchedulerDriver, *mesos.SlaveID) {}
func (s *minerScheduler) ExecutorLost(sched.SchedulerDriver, *mesos.ExecutorID, *mesos.SlaveID, int) {
}

func (s *minerScheduler) Error(driver sched.SchedulerDriver, err string) {
}

func printUsage() {
	println("Usage: scheduler [--FLAGS] [RPC username] [RPC password]")
	println("Your RPC username and password can be found in your bitcoin.conf file.")
	println("To see a detailed description of the flags available, type `scheduler --help`")
}

func main() {
	flag.Parse()

	var user, pass string
	if flag.NArg() == 1 {
		user = ""
		pass = flag.Arg(0)
	} else if flag.NArg() == 2 {
		user = flag.Arg(0)
		pass = flag.Arg(1)
	} else {
		printUsage()
		println(flag.NArg())
		return
	}

	fwinfo := &mesos.FrameworkInfo{
		User: proto.String(""),
		Name: proto.String("BTC Mining Framework (Go)"),
	}

	cred := (*mesos.Credential)(nil)
	if *mesosAuthPrincipal != "" {
		fwinfo.Principal = proto.String(*mesosAuthPrincipal)
		secret, err := ioutil.ReadFile(*mesosAuthSecretFile)
		if err != nil {
			log.Fatal(err)
		}
		cred = &mesos.Credential{
			Principal: proto.String(*mesosAuthPrincipal),
			Secret:    secret,
		}
	}
	config := sched.DriverConfig{
		Scheduler:  newMinerScheduler(user, pass),
		Framework:  fwinfo,
		Master:     *master,
		Credential: cred,
		WithAuthContext: func(ctx context.Context) context.Context {
			ctx = auth.WithLoginProvider(ctx, *authProvider)
			return ctx
		},
	}

	driver, err := sched.NewMesosSchedulerDriver(config)

	if err != nil {
		log.Errorln("Unable to create a SchedulerDriver ", err.Error())
	}

	if stat, err := driver.Run(); err != nil {
		log.Infof("Framework stopped with status %s and error: %s\n", stat.String(), err.Error())
	}
}
