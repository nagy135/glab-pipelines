package main

import "testing"

func TestBuildDisplayJobsNewRunSupersedesManualJob(t *testing.T) {
	rows := buildDisplayJobs([]job{
		{ID: 10, Name: "deploy", Stage: "deploy", Status: "manual"},
		{ID: 11, Name: "deploy", Stage: "deploy", Status: "running"},
	})
	if len(rows) != 1 || rows[0].Current.ID != 11 {
		t.Fatalf("current job = %+v, want job 11", rows)
	}
}
