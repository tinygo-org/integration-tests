package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v31/github"
)

func authenticateGithubClient(appid, installid int64, privatekeyfile string) (*github.Client, error) {
	tr := http.DefaultTransport
	itr, err := ghinstallation.NewKeyFromFile(tr, appid, installid, "keys/"+privatekeyfile)
	if err != nil {
		return nil, err
	}

	return github.NewClient(&http.Client{Transport: itr}), nil
}

func (build Build) pendingCheckSuite() {
	log.Printf("Github check suite pending for %s\n", build.sha)
	for _, board := range boards {
		build.pendingCheckRun(board.target)
	}
}

func (build Build) pendingCheckRun(target string) {
	log.Printf("Github check run pending on board %s for %s\n", target, build.sha)
	opts := github.CreateCheckRunOptions{
		Name:    targetName(target),
		HeadSHA: build.sha,
	}
	cr, _, err := client.Checks.CreateCheckRun(context.Background(), ghorg, ghrepo, opts)
	if err != nil {
		log.Println(err)
	}
	build.runs[target] = cr
}

func (build Build) startCheckSuite() {
	log.Printf("Github check suite starting for %s\n", build.sha)
	for _, run := range build.runs {
		target, _ := parseTarget(run.GetName())
		build.startCheckRun(target)
	}
}

func (build Build) startCheckRun(target string) {
	log.Printf("Github check run starting on board %s for %s\n", target, build.sha)
	status := "in_progress"
	if run, ok := build.runs[target]; ok {
		opts := github.UpdateCheckRunOptions{
			Name:   targetName(target),
			Status: &status,
		}
		cr, _, err := client.Checks.UpdateCheckRun(context.Background(), ghorg, ghrepo, *run.ID, opts)
		if err != nil {
			log.Println(err)
		}
		build.runs[target] = cr
	}
}

func (build Build) passCheckRun(target, output string) {
	log.Printf("Github check run passed on board %s for %s\n", target, build.sha)
	title := "Hardware CI passed"
	summary := "Hardware CI tests have passed."
	status := "completed"
	conclusion := "success"
	timestamp := github.Timestamp{Time: time.Now()}
	if run, ok := build.runs[target]; ok {
		ro := github.CheckRunOutput{
			Title:   &title,
			Summary: &summary,
			Text:    &output,
		}

		opts := github.UpdateCheckRunOptions{
			Name:        targetName(target),
			Status:      &status,
			Conclusion:  &conclusion,
			CompletedAt: &timestamp,
			Output:      &ro,
		}
		_, _, err := client.Checks.UpdateCheckRun(context.Background(), ghorg, ghrepo, *run.ID, opts)
		if err != nil {
			log.Println(err)
		}
		delete(build.runs, target)
	}
}

func (build Build) failCheckSuite(output string) {
	log.Printf("Github check suite failed for %s\n", build.sha)
	for _, run := range build.runs {
		target, _ := parseTarget(run.GetName())
		build.failCheckRun(target, output)
	}
}

func (build Build) failCheckRun(target, output string) {
	log.Printf("Github check run failed on board %s for %s\n", target, build.sha)
	title := "Hardware CI failed"
	summary := "Hardware CI tests have failed."
	status := "completed"
	conclusion := "failure"
	timestamp := github.Timestamp{Time: time.Now()}
	if run, ok := build.runs[target]; ok {
		ro := github.CheckRunOutput{
			Title:   &title,
			Summary: &summary,
			Text:    &output,
		}

		opts := github.UpdateCheckRunOptions{
			Name:        targetName(target),
			Status:      &status,
			Conclusion:  &conclusion,
			CompletedAt: &timestamp,
			Output:      &ro,
		}
		_, _, err := client.Checks.UpdateCheckRun(context.Background(), ghorg, ghrepo, *run.ID, opts)
		if err != nil {
			log.Println(err)
		}
		delete(build.runs, target)
	}
}

// reload the check runs from github for this build
func (build Build) reloadCheckRuns() error {
	opts := github.ListCheckRunsOptions{}
	res, _, err := client.Checks.ListCheckRunsForRef(context.Background(), ghorg, ghrepo, build.sha, &opts)
	if err != nil {
		return err
	}

	for _, run := range res.CheckRuns {
		if !strings.Contains(run.GetName(), "tinyhci:") {
			continue
		}

		target, err := parseTarget(run.GetName())
		if err != nil {
			return err
		}
		build.runs[target] = run
	}

	return nil
}

// reload the check runs from github for this build
func findQueuedCheckRuns(sha string) ([]*github.CheckRun, error) {
	runs := make([]*github.CheckRun, 0)
	status := "queued"
	opts := github.ListCheckRunsOptions{Status: &status}
	res, _, err := client.Checks.ListCheckRunsForRef(context.Background(), ghorg, ghrepo, sha, &opts)
	if err != nil {
		return nil, err
	}

	for _, run := range res.CheckRuns {
		if !strings.Contains(run.GetName(), "tinyhci:") {
			continue
		}

		fmt.Println(run.GetID(), run.GetStatus())
		runs = append(runs, run)
	}

	return runs, nil
}

func targetName(target string) string {
	return "tinyhci: " + target
}

func parseTarget(name string) (string, error) {
	res := strings.Split(name, " ")
	if len(res) != 2 {
		return "", errors.New("invalid check run name")
	}
	return res[1], nil
}
