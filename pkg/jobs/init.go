package jobs

import (
	"fmt"
	"myproject/models"
	"os/exec"
	"time"
	"bytes"
	"myproject/pkg/requests"
	"myproject/pkg/logging"
	"go.uber.org/zap"
)

func InitJobs() {
	list,errinit:= models.TaskInitList(1, 1000000, 1)
	if errinit != nil{
		logging.AppLogger.Fatal("初始化定时任务失败", zap.Error(errinit))
	}
	for _, task := range list {
		job, err := NewJobFromTask(task)
		if err != nil {
			continue
		}
		AddJob(task.CronSpec, job)
	}
}

func runCmdWithTimeout(cmd *exec.Cmd, timeout time.Duration) (error, bool) {
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	var err error
	select {
	case <-time.After(timeout):
		logging.AppLogger.Warn(fmt.Sprintf("任务执行时间超过%d秒，进程将被强制杀掉: %d", int(timeout/time.Second), cmd.Process.Pid))
		// fmt.Print(fmt.Sprintf("任务执行时间超过%d秒，进程将被强制杀掉: %d", int(timeout/time.Second), cmd.Process.Pid))
		go func() {
			<-done // 读出上面的goroutine数据，避免阻塞导致无法退出
		}()
		if err = cmd.Process.Kill(); err != nil {
			logging.AppLogger.Warn(fmt.Sprintf("进程无法杀掉: %d, 错误信息: %s", cmd.Process.Pid, err))
			// fmt.Print(fmt.Sprintf("进程无法杀掉: %d, 错误信息: %s", cmd.Process.Pid, err))
		}
		return err, true
	case err = <-done:
		return err, false
	}
}

func Interprete(timeout time.Duration, task *models.Task) (string, string, error, bool){
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	var cmd *exec.Cmd
	switch task.TaskType {
		case models.SHELL:
			cmd = exec.Command("bash", "-c", task.Command)
		case models.PYTHON:
			cmd = exec.Command("python3", task.Command)
		default:
			cmd = exec.Command("bash", "-c", task.Command)
	}
	cmd.Stdout = bufOut
	cmd.Stderr = bufErr
	cmd.Start()
	err, isTimeout := runCmdWithTimeout(cmd, timeout)

	return bufOut.String(), bufErr.String(), err, isTimeout
}

func HttpGet(timeout time.Duration, task *models.Task) (string, string, error, bool){
	return requests.Get(task.Command,timeout)
}