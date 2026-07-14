package main

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
)

const (
	modePipelines = iota
	modeDetail
	modeJobs
	modeConfirm
	modeLogs
	modeTheme
	modeCode
)

const (
	logSplitHorizontal = iota
	logSplitVertical
)

var activeStatuses = []string{"running", "pending", "created", "waiting_for_resource", "preparing", "manual", "scheduled"}

type pipeline struct {
	ID          int        `json:"id"`
	Status      string     `json:"status"`
	Ref         string     `json:"ref"`
	SHA         string     `json:"sha"`
	Source      string     `json:"source"`
	UpdatedAt   string     `json:"updated_at"`
	CreatedAt   string     `json:"created_at"`
	StartedAt   string     `json:"started_at"`
	Duration    *float64   `json:"duration"`
	WebURL      string     `json:"web_url"`
	Commit      commitInfo `json:"commit"`
	CommitTitle string     `json:"commit_title,omitempty"`
}

type commitInfo struct {
	Title string `json:"title"`
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
	Job        job
	PipelineID int
	Endpoint   string
	Verb       string
}

type logSearchMatch struct {
	Line  int
	Start int
	End   int
}

type inlineLogSnippet struct {
	Lines  []string
	Status string
}

type logPane struct {
	ID               int
	Mode             int
	ListCursor       int
	DetailID         int
	Detail           *detail
	JobsCursor       int
	Job              *job
	BackMode         int
	Logs             string
	Loading          bool
	Viewport         viewport.Model
	SearchMode       bool
	SearchActive     bool
	SearchQuery      string
	SearchMatches    []logSearchMatch
	SearchIndex      int
	ShowInlineLogs   bool
	WrapContent      bool
	ShowLineNumbers  bool
	ScrollOffset     int
	HorizontalOffset int
}

type logSplitNode struct {
	PaneID    int
	Direction int
	First     *logSplitNode
	Second    *logSplitNode
}

type model struct {
	repo              string
	status            string
	limit             int
	refresh           time.Duration
	logRefresh        time.Duration
	mode              int
	width             int
	height            int
	activity          spinner.Model
	message           string
	loadingList       bool
	listRequest       int
	nextRequestID     int
	detailRequests    map[int]int
	detailPolls       map[int]int
	logRequests       map[int64]int
	logPolls          map[int64]int
	logFailures       map[int64]int
	codeRequests      map[int64]int
	list              []pipeline
	listCursor        int
	detailID          int
	detail            *detail
	detailLoading     bool
	jobsCursor        int
	pending           *pendingAction
	actionInFlight    bool
	actionRequest     int
	confirmText       string
	confirmBackMode   int
	logJob            *job
	logBackMode       int
	logs              string
	logsLoading       bool
	logsViewport      viewport.Model
	logSearchMode     bool
	logSearchActive   bool
	logSearchQuery    string
	logSearchMatches  []logSearchMatch
	logSearchIndex    int
	showInlineLogs    bool
	wrapContent       bool
	showLineNumbers   bool
	inlineLogs        map[int64]inlineLogSnippet
	inlineLogRequests map[int64]int
	inlineLogsLoading map[int64]bool
	inlineLogPollID   int
	logPanes          []logPane
	activeLogPane     int
	nextLogPaneID     int
	logSplitRoot      *logSplitNode
	jobStatuses       map[int64]string
	jobDurations      map[string]jobDurationStat
	themeName         string
	themeCursor       int
	themeBackMode     int
	borderName        string
	scrollOffset      int
	horizontalOffset  int
}

type pipelinesMsg struct {
	requestID int
	pipelines []pipeline
	err       error
}

type detailMsg struct {
	pid       int
	requestID int
	pollID    int
	detail    detail
	err       error
}

type actionMsg struct {
	requestID int
	action    pendingAction
	err       error
}

type logsMsg struct {
	jobID     int64
	requestID int
	pollID    int
	logs      string
	job       *job
	err       error
	statusErr error
}

type codeMsg struct {
	jobID     int64
	requestID int
	code      string
	err       error
}

type tickMsg struct {
	pid    int
	pollID int
}

type logTickMsg struct {
	jobID  int64
	pollID int
	force  bool
}

type inlineLogMsg struct {
	jobID     int64
	requestID int
	pollID    int
	status    string
	lines     []string
	err       error
}

type inlineLogTickMsg struct {
	pollID int
}
