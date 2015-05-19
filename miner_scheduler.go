package main

import (
	"strconv"

	"github.com/gogo/protobuf/proto"
	log "github.com/golang/glog"

	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
)

const (
	MEM_PER_TASK = 128 // mining shouldn't be memory-intensive
)

type MinerScheduler struct {
	executor *mesos.ExecutorInfo
}

func newMinerScheduler(exec *mesos.ExecutorInfo) *MinerScheduler {
	return &MinerScheduler{
		executor: exec,
	}
}

func (s *MinerScheduler) Registered(driver sched.SchedulerDriver, frameworkId *mesos.FrameworkID, masterInfo *mesos.MasterInfo) {
	log.Infoln("Framework registered with Master ", masterInfo)
}

func (s *MinerScheduler) Reregistered(driver sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {
	log.Infoln("Framework Re-Registered with Master ", masterInfo)
}

func (s *MinerScheduler) Disconnected(sched.SchedulerDriver) {
	log.Infoln("Framework disconnected with Master ", masterInfo)
}

func (s *MinerScheduler) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	for i, offer := range offers {
		memResources := util.FilterResources(offer.Resources, func(res *mesos.Resource) bool {
			return res.GetName() == "mem"
		})
		mems := 0.0
		for _, res := range memResources {
			mems += res.GetScalar().GetValue()
		}
		if mems < MEM_PER_TASK {
			continue
		}

		cpuResources := util.FilterResources(offer.Resources, func(res *mesos.Resource) bool {
			return res.GetName() == "cpus"
		})
		cpus := 0.0
		for _, res := range cpuResources {
			cpus += res.GetScalar().GetValue()
		}

		taskId := &mesos.TaskID{
			Value: proto.String(strconv.Itoa(i)),
		}

		task := &mesos.TaskInfo{
			Name:     proto.String("miner-task-" + taskID.GetValue()),
			TaskId:   taskId,
			SlaveId:  offer.SlaveId,
			Executor: sched.executor,
			Command:  &mesos.CommandInfo{},
			Resources: []*mesos.Resource{
				util.NewScalarResource("cpus", cpus),
				util.NewScalarResource("mem", MEM_PER_TASK),
			},
		}
		log.Infof("Prepared task: %s with offer %s for launch\n", task.GetName(), offer.Id.GetValue())
	}
}
