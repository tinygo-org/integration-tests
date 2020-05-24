package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"

	"net/http"

	"github.com/google/go-github/v31/github"
)

const (
	useCurrentBinaryRelease = false // set to true to use the already installed tinygo
	officialRelease         = "https://github.com/tinygo-org/tinygo/releases/download/v0.13.1/tinygo0.13.1.linux-amd64.tar.gz"
)

var (
	// these will be overwritten by the ENV vars of the same name
	ghorg  = "tinygo-org"
	ghrepo = "tinygo"

	ghwebhookpath = "/webhooks"
	ciwebhookpath = "/buildhook"

	client *github.Client

	// key is sha
	builds map[string]*Build

	pollFrequency = 30 * time.Second
)

func main() {
	ghwebhookpath = os.Getenv("GHWEBHOOKPATH")
	if ghwebhookpath == "" {
		log.Fatal("You must set an ENV var with your GHWEBHOOKPATH")
	}

	ciwebhookpath = os.Getenv("CIWEBHOOKPATH")
	if ciwebhookpath == "" {
		log.Fatal("You must set an ENV var with your CIWEBHOOKPATH")
	}

	ghorg = os.Getenv("GHORG")
	if ghorg == "" {
		log.Fatal("You must set an ENV var with your GHORG")
	}

	ghrepo = os.Getenv("GHREPO")
	if ghrepo == "" {
		log.Fatal("You must set an ENV var with your GHREPO")
	}

	ghkey := os.Getenv("GHKEY")
	if ghkey == "" {
		log.Fatal("You must set an ENV var with your GHKEY")
	}

	ghkeyfile := os.Getenv("GHKEYFILE")
	if ghkeyfile == "" {
		log.Fatal("You must set an ENV var with your GHKEYFILE")
	}

	aid := os.Getenv("GHAPPID")
	if aid == "" {
		log.Fatal("You must set an ENV var with your GHAPPID")
	}

	iid := os.Getenv("GHINSTALLID")
	if iid == "" {
		log.Fatal("You must set an ENV var with your GHINSTALLID")
	}

	appid, err := strconv.Atoi(aid)
	if err != nil {
		log.Fatal("Invalid Github app id")
	}

	installid, err := strconv.Atoi(iid)
	if err != nil {
		log.Fatal("Invalid Github install id")
	}

	client, err = authenticateGithubClient(int64(appid), int64(installid), ghkeyfile)
	if err != nil {
		log.Println(err)
	}

	builds = make(map[string]*Build)
	buildsCh := make(chan *Build)

	handlePreviouslyQueuedBuilds(buildsCh)

	go processBuilds(buildsCh)
	go pollPendingBuilds(buildsCh)

	http.HandleFunc(ghwebhookpath, func(w http.ResponseWriter, r *http.Request) {
		payload, err := github.ValidatePayload(r, []byte(ghkey))
		if err != nil {
			log.Println("Invalid webhook payload")
			return
		}
		event, err := github.ParseWebHook(github.WebHookType(r), payload)
		if err != nil {
			log.Println("Invalid webhook event")
			return
		}
		switch event := event.(type) {
		case *github.PushEvent:
			// ignore pushes because we only care about checks API
			return
		case *github.CheckSuiteEvent:
			log.Printf("Github checksuite event %s %s for %d %s\n",
				event.CheckSuite.GetStatus(),
				event.CheckSuite.GetConclusion(),
				event.CheckSuite.GetID(),
				event.CheckSuite.GetHeadSHA())

			switch event.CheckSuite.GetStatus() {
			case "completed":
				// just in case we want to do something here
			case "queued":
				// received when a new commit is pushed
				build := NewBuild(event.CheckSuite.GetHeadSHA())
				build.pendingCI = true
				build.started = time.Now()
				builds[build.sha] = build
				build.pendingCheckSuite()
			default:
			}

		case *github.CheckRunEvent:
			log.Printf("Github checkrun event %s %s for %d %s %s %s\n",
				event.CheckRun.GetStatus(),
				event.CheckRun.GetConclusion(),
				event.CheckRun.GetID(),
				event.CheckRun.GetName(),
				event.GetAction(),
				event.CheckRun.GetHeadSHA())

			switch event.CheckRun.GetStatus() {
			case "completed":
				if event.GetAction() == "rerequested" {
					performCheckRun(event.CheckRun, buildsCh)
				}
			case "queued":
				// received when a new commit is pushed
			default:
			}

		default:
			log.Println("Unexpected Github event:", event)
		}
	})

	// we can remove this soon.
	http.HandleFunc(ciwebhookpath, func(w http.ResponseWriter, r *http.Request) {
		log.Println("CircleCI buildhook received.")
		bi, err := parseBuildInfo(r)
		if err != nil {
			log.Println(err)
			return
		}

		log.Printf("Build Info: %+v\n", bi)
	})

	log.Printf("Starting TinyHCI server for %s/%s\n", ghorg, ghrepo)
	http.ListenAndServe(":8000", nil)
}

// processBuilds is run as a go routine to pull new builds
// from the build channel, and then perform the needed build
// tasks aka build docker image, then flash/test for each board.
func processBuilds(builds chan *Build) {
	for {
		select {
		case build := <-builds:
			log.Printf("Starting tests for commit %s\n", build.sha)
			build.startCheckSuite()

			url := officialRelease
			if !useCurrentBinaryRelease {
				url = build.binaryURL
			}

			log.Printf("Building docker image using TinyGo from %s\n", url)
			err := buildDocker(url, build.sha)
			if err != nil {
				log.Println(err)
				build.failCheckSuite("docker build failed")
				continue
			}

			log.Printf("Running checks for commit %s\n", build.sha)
			for _, run := range build.runs {
				target, _ := parseTarget(run.GetName())
				board := GetBoard(target)
				build.processBoardRun(board)
			}
		}
	}
}

// buildDocker does the docker build for the binary download
// with this SHA.
func buildDocker(url, sha string) error {
	buildarg := fmt.Sprintf("TINYGO_DOWNLOAD_URL=%s", url)
	buildtag := "tinygohci:" + sha[:7]
	out, err := exec.Command("docker", "build",
		"-t", buildtag,
		"-f", "tools/docker/Dockerfile",
		"--build-arg", buildarg, ".").CombinedOutput()
	if err != nil {
		log.Println(err)
		log.Println(string(out))
		return err
	}

	return nil
}

// pollPendingBuilds is run as a go routine to poll the
// circleci server, and look for new builds of the TinyGo
// binary that match tinyhci builds in need of processing.
func pollPendingBuilds(buildsCh chan *Build) {
	for {
		if len(builds) == 0 {
			log.Println("No builds to poll for.")
		} else {
			log.Println("Polling for builds...")
		}

		for _, b := range builds {
			if b.pendingCI {
				// look to see if there is a CI build with binary for this build
				bn, err := getMostRecentCIBuildNumAfterStart(b.sha, b.started)
				if err != nil {
					continue
				}
				log.Println("Binary ready for", b.sha)

				url, err := getTinygoBinaryURL(bn)
				if err != nil {
					log.Println(err)
					continue
				}

				b.binaryURL = url
				b.pendingCI = false

				buildsCh <- b
			}
		}

		// sleep in-between waiting for new builds
		time.Sleep(pollFrequency)
	}
}

func performCheckRun(cr *github.CheckRun, buildsCh chan *Build) {
	// do the retest here
	bn, err := getCIBuildNumFromSHA(cr.GetHeadSHA())
	if err != nil {
		log.Println(err)
		return
	}

	url, err := getTinygoBinaryURL(bn)
	if err != nil {
		log.Println(err)
		return
	}

	target, err := parseTarget(cr.GetName())
	if err != nil {
		log.Println(err)
		return
	}

	build := NewBuild(cr.GetHeadSHA())
	build.binaryURL = url
	build.runs[target] = cr
	builds[build.sha] = build

	// handoff to channel for processing
	buildsCh <- build
}

// handlePreviouslyQueuedBuilds retrieves builds that were
// already queued before the server was started, probably due
// to some error or failure.
func handlePreviouslyQueuedBuilds(buildsCh chan *Build) {
	cibuilds, err := getRecentSuccessfulCIBuilds()
	if err != nil {
		log.Println(err)
		return
	}

	for _, cib := range cibuilds {
		// any queued checkruns for this build?
		runs, err := findQueuedCheckRuns(cib.VcsRevision)
		if err != nil {
			log.Println(err)
			return
		}

		for _, run := range runs {
			performCheckRun(run, buildsCh)
		}
	}
}
