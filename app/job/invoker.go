package job

import (
	"github.com/shotdog/quartz"
	"kitty/app/service"
	"time"
	"kitty/app/model"
	"log"
	"net/http"
	"encoding/json"
	"kitty/app/common"
	"bytes"
)

var JobManager *jobManager

type jobManager struct {
	qz *quartz.Quartz
}

func NewJobManager() {

	if JobManager == nil {

		qz := quartz.New()
		qz.BootStrap()
		JobManager = &jobManager{qz:qz}
	}

}

func (this *jobManager)PushAllJob() {

	list, err := service.JobInfoService.List()
	if err != nil || len(list) == 0 {
		return
	}

	for _, jobInfo := range list {

		this.AddJob(jobInfo)

	}

}

func (this *jobManager)AddJob(jobInfo model.JobInfo) error {

	return this.qz.AddJob(&quartz.Job{
		Id:jobInfo.Id,
		Name:jobInfo.JobName,
		Group:jobInfo.JobGroup,
		Url:jobInfo.Url,
		Params:jobInfo.Params,
		Expression:jobInfo.Cron,
		JobFunc:invoke,

	})

}

// modify
func (this *jobManager)ModifyJob(jobInfo *model.JobInfo) error {

	return this.qz.ModifyJob(&quartz.Job{
		Id:jobInfo.Id,
		Name:jobInfo.JobName,
		Group:jobInfo.JobGroup,
		Url:jobInfo.Url,
		Params:jobInfo.Params,
		Expression:jobInfo.Cron,
		JobFunc:invoke,

	})
}

// remove
func (this *jobManager)RemoveJob(jobInfo model.JobInfo) error {

	return this.qz.RemoveJob(jobInfo.Id)

}

func invoke(jobId int, targetUrl, params string, nextTime time.Time) {
	jobInfo, err := service.JobInfoService.FindJobInfoById(jobId)
	if err != nil || jobInfo.Active == 0 {
		JobManager.RemoveJob(jobInfo)
		return
	}

	//

	initExecute(jobInfo, targetUrl, nextTime)

}

func initExecute(jobInfo model.JobInfo, targetUrl string, nextTime time.Time) {

	snapshot := &model.JobSnapshot{
		JobName:jobInfo.JobName,
		JobGroup:jobInfo.JobGroup,
		Cron:jobInfo.Cron,
		Url:jobInfo.Url,
		JobId:jobInfo.Id,
		Detail:"【" + time.Now().Format("2006-01-02 15:04:05") + "】初始化完成目标服务器:" + targetUrl,
		CreateTime:time.Now(),
		State:0,

	}

	err := service.JobSnapshotService.Add(snapshot)
	if err != nil {
		return
	}

	log.Println("snapshot:", snapshot)

	invokeJob(snapshot)

}

func invokeJob(snapshot *model.JobSnapshot) {

	detail := snapshot.Detail + "\n【" + time.Now().Format("2006-01-02 15:04:05") + "】正在调用..."
	err := service.JobSnapshotService.Update(snapshot.Id, 1, detail, time.Now())
	if err != nil {
		return
	}

	req := common.Request{
		SnapshotId:snapshot.Id,
		JobId:snapshot.JobId,
		Params     :snapshot.Params,
		Method     :"INVOKE",
	}
	body,_:=json.Marshal(req)

	res, err := http.Post(snapshot.Url,"application/json;charset=utf-8",bytes.NewReader(body))
	if err != nil {

		detail = detail + "\n【" + time.Now().Format("2006-01-02 15:04:05") + "】目标服务器不可用..."
		service.JobSnapshotService.Update(snapshot.Id, 4, detail, time.Now())
		return

	} else {

		res.Body.Close()
	}

	log.Println("snapshot:", snapshot)

}

type JobInvoker struct {

}
