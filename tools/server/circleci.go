package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/metanerd/go-circleci"
)

// CIBuildInfo is info from CircleCI webhook sent by the CircleCI Webhook Orb
// https://github.com/eddiewebb/circleci-webhook-orb
type CIBuildInfo struct {
	BuildNum    string `json:"build_num"`
	Branch      string `json:"branch"`
	Username    string `json:"username"`
	Job         string `json:"job"`
	BuildURL    string `json:"build_url"`
	VCSRevision string `json:"vcs_revision"`
	RepoName    string `json:"reponame"`
	WorkflowID  string `json:"workflow_id"`
	WorkflowURL string `json:"workflow_url"`
	PullRequest string `json:"pull_request"`
	User        string `json:"user"`
	APILink     string `json:"api_link"`
	Status      string `json:"status"`
}

func parseBuildInfo(r *http.Request) (*CIBuildInfo, error) {
	// Read body
	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		return nil, err
	}

	// Unmarshal
	var bi CIBuildInfo
	err = json.Unmarshal(b, &bi)
	if err != nil {
		return nil, err
	}

	return &bi, nil
}

func getTinygoBinaryURL(buildNum string) (string, error) {
	if useCurrentBinaryRelease {
		return "using current TinyGo binary release", nil
	}

	client := &circleci.Client{}
	bn, err := strconv.Atoi(buildNum)
	if err != nil {
		return "", errors.New("invalid buildnum: " + buildNum)
	}
	artifacts, _ := client.ListBuildArtifacts("github", ghorg, ghrepo, bn)

	for _, a := range artifacts {
		// we're looking for the .tar.gz file
		if a.Path == "tmp/tinygo.linux-amd64.tar.gz" {
			return a.URL, nil
		}
	}
	return "", errors.New("cannot find TinyGo build artifact file")
}

func getCIBuildNumFromSHA(sha string) (string, error) {
	if useCurrentBinaryRelease {
		return "using current TinyGo binary release", nil
	}

	client := &circleci.Client{}

	for offset := 0; offset <= 500; offset += 100 {
		cibuilds, err := client.ListRecentBuildsForProject("github", ghorg, ghrepo, "", "", 100, offset)
		if err != nil {
			return "", err
		}

		for _, b := range cibuilds {
			// we're looking for the sha
			if b.BuildParameters["CIRCLE_JOB"] == "build-linux" &&
				b.VcsRevision == sha &&
				b.Status == "success" {

				bn := strconv.Itoa(b.BuildNum)
				return bn, nil
			}
		}
	}

	return "", fmt.Errorf("cannot find TinyGo build for %s", sha)
}

func getMostRecentCIBuildNumAfterStart(sha string, start time.Time) (string, error) {
	if useCurrentBinaryRelease {
		return "using current TinyGo binary release", nil
	}

	client := &circleci.Client{}

	for offset := 0; offset <= 500; offset += 100 {
		cibuilds, err := client.ListRecentBuildsForProject("github", ghorg, ghrepo, "", "", 100, offset)
		if err != nil {
			return "", err
		}

		for _, b := range cibuilds {
			if b.BuildParameters["CIRCLE_JOB"] == "build-linux" &&
				start.Before(*b.StartTime) &&
				b.VcsRevision == sha &&
				b.Status == "success" {

				bn := strconv.Itoa(b.BuildNum)
				return bn, nil
			}
		}
	}

	return "", fmt.Errorf("cannot find TinyGo build for %s", sha)
}

func getRecentSuccessfulCIBuilds() ([]*circleci.Build, error) {
	successes := make([]*circleci.Build, 0)
	client := &circleci.Client{}

	builds, err := client.ListRecentBuildsForProject("github", "tinygo-org", "tinygo", "", "", 100, 0)
	if err != nil {
		return nil, err
	}

	for _, build := range builds {
		if build.BuildParameters["CIRCLE_JOB"] != "build-linux" &&
			build.Status != "success" {
			continue
		}

		successes = append(successes, build)
	}

	return successes, nil
}
