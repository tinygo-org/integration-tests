package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type Board struct {
	target      string
	displayname string
	port        string
	baud        int
	resetpause  time.Duration
}

var (
	boards = []*Board{
		&Board{
			target:      "itsybitsy-m4",
			displayname: "Adafruit ItsyBitsy-M4",
			port:        "itsybitsy_m4",
			baud:        115200,
			resetpause:  2 * time.Second,
		},
		&Board{
			target:      "arduino",
			displayname: "Arduino Uno",
			port:        "arduino_uno",
			baud:        57600,
			resetpause:  5 * time.Second,
		},
		&Board{
			target:      "arduino-nano33",
			displayname: "Arduino Nano33 IoT",
			port:        "arduino_nano33",
			baud:        115200,
			resetpause:  2 * time.Second,
		},
		&Board{
			target:      "microbit",
			displayname: "bbc:microbit",
			port:        "microbit",
			baud:        115200,
			resetpause:  2 * time.Second,
		},
	}
)

// GetBoard returns the board for this target.
func GetBoard(target string) *Board {
	for _, b := range boards {
		if b.target == target {
			return b
		}
	}
	return nil
}

func (board *Board) flash(sha string) (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return err.Error(), err
	}
	buildtag := fmt.Sprintf("tinygohci:%s", sha[:7])
	device := fmt.Sprintf("--device=/dev/%s", board.port)
	port := fmt.Sprintf("-port=/dev/%s", board.port)
	file := fmt.Sprintf("/src/%s/main.go", board.target)
	out, err := exec.Command("docker", "run",
		device,
		"-v", "/media:/media:shared",
		"-v", pwd+":/src",
		"--rm",
		buildtag,
		"tinygo", "flash",
		"-size", "short",
		"-target", board.target,
		port,
		file).CombinedOutput()
	return string(out), err
}

func (board *Board) test() (string, error) {
	port := fmt.Sprintf("/dev/%s", board.port)
	br := strconv.Itoa(board.baud)

	out, err := exec.Command("./build/testrunner", port, br, "5").CombinedOutput()
	return string(out), err
}
