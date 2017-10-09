package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"golang.org/x/oauth2/google"
	bq "google.golang.org/api/bigquery/v2"
)

var (
	ctx = context.Background()
)

// BQService wraps low level BigQuery API and exposes convenient methods
type BQService struct {
	projectID   string
	jobsService *bq.JobsService
}

// NewBQService create an instance of BQService for a specific
// project, it requires application default credentials obtained by
// background context
func NewBQService(projectID string) *BQService {
	httpClient, err := google.DefaultClient(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	bqs, err := bq.New(httpClient)
	service := bq.NewJobsService(bqs)

	return &BQService{projectID: projectID, jobsService: service}
}

// GetJobs returns jobs in reverse chronological order of their execution,
// i.e. running or most recently ran  jobs first
// pageToken may be passed to iterate order jobs
func (b *BQService) GetJobs(pageToken string) *Jobs {
	call := b.jobsService.List(b.projectID).AllUsers(true).Projection("full")
	if pageToken != "" {
		call = call.PageToken(pageToken)
	}
	jobsList, err := call.Do()
	if err != nil {
		log.Fatalf("unable to list running jobs %v", err)
	}

	jobs := &Jobs{}

	jobs.NextPage = jobsList.NextPageToken
	for _, j := range jobsList.Jobs {
		job := b.parseJobListJobs(j)
		if job.Status == "RUNNING" {
			jobs.Running = append(jobs.Running, job)
		} else {
			jobs.Done = append(jobs.Done, job)
		}
	}

	return jobs
}

// GetJob returns a single job value with specific information about that job
func (b *BQService) GetJob(jobID string) *Job {
	j, err := b.jobsService.Get(b.projectID, jobID).Do()
	if err != nil {
		log.Fatal(err)
	}
	job := b.parseJob(j)
	return job
}

// CancelJob attempts to cancel a running job
func (b *BQService) CancelJob(jobID string) {
	cancelCall, err := b.jobsService.Cancel(b.projectID, jobID).Do()
	if err != nil {
		log.Fatal(err)
	}
	log.Println(*cancelCall)
}

func (b *BQService) parseJob(bqJob *bq.Job) *Job {
	jobID := strings.Split(bqJob.Id, ":")[1]
	job := Job{
		ID:               jobID,
		UserName:         bqJob.UserEmail,
		DataQueried:      bqJob.Statistics.TotalBytesProcessed,
		HumanDataQueried: bytefmt.ByteSize(uint64(bqJob.Statistics.TotalBytesProcessed)),
		Status:           bqJob.Status.State,
		StartTime:        time.Unix(0, bqJob.Statistics.StartTime*int64(time.Millisecond)),
	}
	cost := float64(bqJob.Statistics.TotalBytesProcessed) / math.Pow(1024, 4) * 5.0
	job.QueryCost = fmt.Sprintf("$%.5f", cost)
	if bqJob.Configuration != nil {
		if bqJob.Configuration.Query != nil {
			job.Query = bqJob.Configuration.Query.Query
		}
	}

	if job.Status == "DONE" {
		job.EndTime = time.Unix(0, bqJob.Statistics.EndTime*int64(time.Millisecond))
		job.RunTime = job.EndTime.Sub(job.StartTime)
	} else {
		job.RunTime = time.Now().Sub(job.StartTime)
	}
	return &job
}
func (b *BQService) parseJobListJobs(bqJob *bq.JobListJobs) *Job {
	jobID := strings.Split(bqJob.Id, ":")[1]
	job := Job{
		ID:               jobID,
		UserName:         bqJob.UserEmail,
		DataQueried:      bqJob.Statistics.TotalBytesProcessed,
		HumanDataQueried: bytefmt.ByteSize(uint64(bqJob.Statistics.TotalBytesProcessed)),
		Status:           bqJob.Status.State,
		StartTime:        time.Unix(0, bqJob.Statistics.StartTime*int64(time.Millisecond)),
	}

	cost := float64(bqJob.Statistics.TotalBytesProcessed) / math.Pow(1024, 4) * 5.0
	job.QueryCost = fmt.Sprintf("$%.5f", cost)
	if bqJob.Configuration != nil {
		if bqJob.Configuration.Query != nil {
			job.Query = bqJob.Configuration.Query.Query
		}
	}

	if job.Status == "DONE" {
		job.EndTime = time.Unix(0, bqJob.Statistics.EndTime*int64(time.Millisecond))
		job.RunTime = job.EndTime.Sub(job.StartTime)
	} else {
		job.RunTime = time.Now().Sub(job.StartTime)
	}
	return &job
}

// Job holds information about a single bigquery job
type Job struct {
	ID               string
	UserName         string
	Query            string
	QueryCost        string
	DataQueried      int64
	HumanDataQueried string
	Status           string
	StartTime        time.Time
	EndTime          time.Time
	RunTime          time.Duration
}

// Jobs holds a slice of currently running jobs and completed jobs
// and provides NextPage token to be passed to GetJobs method
type Jobs struct {
	Running  []*Job
	Done     []*Job
	NextPage string
}
