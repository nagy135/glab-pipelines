package main

import (
	"time"

	"github.com/charmbracelet/bubbles/viewport"
)

const (
	modePipelines = iota
	modeDetail
	modeJobs
	modeConfirm
	modeLogs
)

var activeStatuses = []string{"running", "pending", "created", "waiting_for_resource", "preparing", "manual", "scheduled"}

type pipeline struct {
	ID        int    `json:"id"`
	Status    string `json:"status"`
	Ref       string `json:"ref"`
	SHA       string `json:"sha"`
	Source    string `json:"source"`
	UpdatedAt string `json:"updated_at"`
	CreatedAt string `json:"created_at"`
	WebURL    string `json:"web_url"`
}

type job struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	Status       string   `json:"status"`
	Stage        string   `json:"stage"`
	Ref          string   `json:"ref"`
	CreatedAt    string   `json:"created_at"`
	StartedAt    string   `json:"started_at"`
	FinishedAt   string   `json:"finished_at"`
	Duration     *float64 `json:"duration"`
	AllowFailure bool     `json:"allow_failure"`
	Retried      bool     `json:"retried"`
	Pipeline     pipeline `json:"pipeline"`
}

type uiJob struct {
	Current  job
	Previous *job
}

type detail struct {
	Pipeline    pipeline
	Jobs        []job
	DisplayJobs []uiJob
}

type pendingAction struct {
	Job      job
	Endpoint string
	Verb     string
}

type model struct {
	repo            string
	status          string
	limit           int
	refresh         time.Duration
	logRefresh      time.Duration
	mode            int
	width           int
	height          int
	message         string
	loadingList     bool
	listRequest     int
	list            []pipeline
	listCursor      int
	detailID        int
	detail          *detail
	jobsCursor      int
	pending         *pendingAction
	confirmText     string
	confirmBackMode int
	logJob          *job
	logBackMode     int
	logs            string
	logsLoading     bool
	logsViewport    viewport.Model
}

type pipelinesMsg struct {
	requestID int
	pipelines []pipeline
	err       error
}

type detailMsg struct {
	pid    int
	detail detail
	err    error
}

type actionMsg struct {
	action pendingAction
	err    error
}

type logsMsg struct {
	jobID int64
	logs  string
	err   error
}

type tickMsg struct{ pid int }
type logTickMsg struct{ jobID int64 }
