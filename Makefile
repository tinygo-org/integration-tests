TARGET_GOVERSION := go1.15
GOINSTALLED := $(shell command -v go 2> /dev/null)
CURRENT_GOVERSION  := $(shell go version | awk '{print $$3}')
TARGET_TINYGOVERSION := 0.14.1
TINYGOINSTALLED := $(shell command -v tinygo 2> /dev/null)
CURRENT_TINYGOVERSION  := $(shell tinygo version | awk '{print $$3}')
NOCOLOR := \033[0m
RED     := \033[0;31m
GREEN   := \033[0;32m

all: clean test

test: test-itsybitsy-m4 test-arduino-nano33 test-arduino-uno

test-itsybitsy-m4: build/testrunner
	tinygo flash -size short -target=itsybitsy-m4 -port=/dev/itsybitsy_m4 ./itsybitsy-m4/
	@sleep 2.0s
	@echo "Running tests..."
	./build/testrunner /dev/itsybitsy_m4 115200 5

test-arduino-nano33: build/testrunner
	tinygo flash -size short -target=arduino-nano33 -port=/dev/arduino_nano33 ./arduino-nano33/
	@sleep 2.0s
	@echo "Running tests..."
	./build/testrunner /dev/arduino_nano33 115200 5

test-docker-itsybitsy-m4: build/testrunner
	docker run --device=/dev/itsybitsy_m4 -v /media:/media:shared -v "$(PWD):/src" tinygohci:latest tinygo flash -target itsybitsy-m4  -port=/dev/itsybitsy_m4 /src/itsybitsy-m4/main.go
	@sleep 2.0s
	@echo "Running tests..."
	./build/testrunner /dev/itsybitsy_m4 115200 5

test-arduino-uno: build/testrunner
	tinygo flash -size short -target=arduino -port=/dev/arduino_uno ./arduino/
	@echo "Running tests..."
	./build/testrunner /dev/arduino_uno 57600 5

test-microbit: build/testrunner
	tinygo flash -size short -target=microbit ./microbit/
	@sleep 2.0s
	@echo "Running tests..."
	./build/testrunner /dev/microbit 115200 5

test-hifive: build/testrunner
	tinygo flash -size short -target=hifive1b ./hifive1b/
	@sleep 5.0s
	@echo "Running tests..."
	./build/testrunner /dev/hifive1b 115200 5

test-circuitplay-express: build/testrunner
	tinygo flash -size short -target=circuitplay-express -port=/dev/circuitplay_express ./circuitplay-express/
#	tinygo flash -size short -target=circuitplay-express ./circuitplay-express/
	@sleep 2.0s
	@echo "Running tests..."
	./build/testrunner /dev/circuitplay_express 115200 5
#	./build/testrunner /dev/ttyACM0 115200 5

test-maixbit: build/testrunner
	tinygo flash -size short -target=maixbit -port=/dev/ttyUSB0 ./maixbit/
#	tinygo flash -size short -target=maixbit -port=/dev/maixbit ./maixbit/
	@sleep 2.0s
	@echo "Running tests..."
	./build/testrunner /dev/ttyUSB0 115200 2
#	./build/testrunner /dev/maixbit 115200 2

test-itsybitsy-nrf52840: build/testrunner
#	tinygo flash -size short -target=itsybitsy-nrf52840 -port=/dev/itsybitsy_nrf52840 ./itsybitsy-nrf52840/
	tinygo flash -size short -target=itsybitsy-nrf52840 ./itsybitsy-nrf52840/
	@sleep 2.0s
	@echo "Running tests..."
#	./build/testrunner /dev/itsybitsy_nrf52840 115200 2
	./build/testrunner /dev/ttyACM0 115200 2

update-go:
	@test "$(CURRENT_GOVERSION)" = "$(TARGET_GOVERSION)" && ( echo "$(RED)$(TARGET_GOVERSION) has already been installed$(NOCOLOR)\n" ; exit 1 )
	wget "https://dl.google.com/go/$(TARGET_GOVERSION).linux-amd64.tar.gz" -O /tmp/go.tar.gz
	sudo rm -rf /usr/local/go
	sudo tar -xzf /tmp/go.tar.gz -C /usr/local

install-go:
ifndef GOINSTALLED
	wget "https://dl.google.com/go/$(TARGET_GOVERSION).linux-amd64.tar.gz" -O /tmp/go.tar.gz
	sudo tar -xzf /tmp/go.tar.gz -C /usr/local
	@echo "# add Go to path" >> ~/.bashrc
	@echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
	source ~/.bashrc
endif
	echo "$(GREEN)Go is now installed:$(NOCOLOR)\n"
	go version

install-tinygo:
ifndef TINYGOINSTALLED
	wget "https://github.com/tinygo-org/tinygo/releases/download/v$(TARGET_TINYGOVERSION)/tinygo$(TARGET_TINYGOVERSION).linux-amd64.tar.gz" -O /tmp/tinygo.tar.gz
	tar -xzf /tmp/tinygo.tar.gz -C /usr/local
	@echo "# add TinyGo to path" >> ~/.bashrc
	@echo 'export PATH="$PATH:/usr/local/tinygo/bin\"' >> ~/.bashrc
	source ~/.bashrc
endif
	echo "$(GREEN)TinyGo is now installed:$(NOCOLOR)\n"
	tinygo version

update-tinygo:
	wget "$(TINYGOUPDATE)" -O /tmp/tinygo.tar.gz
	tar -xzf /tmp/tinygo.tar.gz -C /usr/local

install-bossa:
	sudo apt install libreadline-dev libwxgtk3.0-*
	mkdir -p build
	git clone https://github.com/shumatech/BOSSA.git build/BOSSA
	make -C build/BOSSA
	sudo cp build/BOSSA/bin/bossac /usr/local/bin

build/testrunner:
	mkdir -p build
	go build -o build/testrunner tools/testrunner/*

build/tinygohci:
	mkdir -p build
	go build -o build/tinygohci tools/server/*

clean:
	rm -rf build

clean-tinygohci:
	rm -f build/tinygohci

clean-testrunner:
	rm -f build/testrunner

testrunner: clean-testrunner build/testrunner

tinygohci: clean-tinygohci build/tinygohci

install-web-service:
	sudo cp tools/service/tinygohci.service /etc/systemd/system/
	sudo chmod 644 /etc/systemd/system/tinygohci.service

install-ngrok-service:
	sudo cp tools/service/ngrok.service /etc/systemd/system/
	sudo chmod 644 /etc/systemd/system/ngrok.service

start-web-service:
	sudo systemctl start tinygohci
	sudo systemctl enable tinygohci

stop-web-service:
	sudo systemctl stop tinygohci.service

start-ngrok-service:
	sudo systemctl start ngrok
	sudo systemctl enable ngrok

stop-ngrok-service:
	sudo systemctl stop ngrok.service

docker:
	docker build -t tinygohci -f tools/docker/Dockerfile --build-arg TINYGO_DOWNLOAD_URL=https://github.com/tinygo-org/tinygo/releases/download/v0.13.1/tinygo0.13.1.linux-amd64.tar.gz .

powercycle-arduino-nano33:
	@uhubctl -l 1-2.4.4 -a off -p 1
	DEV="/sys/bus/usb/devices/1-2.4.4.1/"; \
	if [ -d $$DEV ]; then \
		sudo udevadm trigger --action=remove $$DEV ; \
	fi
	@sleep 3.0s
	@uhubctl -l 1-2.4.4 -a on -p 1

powercycle-microbit:
	@uhubctl -l 1-2.4.4 -a off -p 2
	DEV="/sys/bus/usb/devices/1-2.4.4.2/"; \
	if [ -d $$DEV ]; then \
		sudo udevadm trigger --action=remove $$DEV ; \
	fi
	@sleep 3.0s
	@uhubctl -l 1-2.4.4 -a on -p 2

powercycle-arduino-uno:
	@uhubctl -l 1-2.4.4 -a cycle -p 3

powercycle-itsybitsy-m4:
	@uhubctl -l 1-2.4.4 -a off -p 4
	DEV="/sys/bus/usb/devices/1-2.4.4.4/"; \
	if [ -d $$DEV ]; then \
		sudo udevadm trigger --action=remove $$DEV ; \
	fi
	@sleep 3.0s
	@uhubctl -l 1-2.4.4 -a on -p 4

powercycle-hifive:
	@uhubctl -l 1-2.4 -a off -p 3
	DEV="/sys/bus/usb/devices/1-2.4.3/"; \
	if [ -d $$DEV ]; then \
		sudo udevadm trigger --action=remove $$DEV ; \
	fi
	@sleep 3.0s
	@uhubctl -l 1-2.4 -a off -p 2
	DEV="/sys/bus/usb/devices/1-2.4.2/"; \
	if [ -d $$DEV ]; then \
		sudo udevadm trigger --action=remove $$DEV ; \
	fi
	@sleep 3.0s
	@uhubctl -l 1-2.4 -a on -p 2
	@sleep 3.0s
	@uhubctl -l 1-2.4 -a on -p 3

powercycle-circuitplay-express:
	@uhubctl -l 1-2.4 -a off -p 1
	DEV="/sys/bus/usb/devices/1-2.4.1/"; \
	if [ -d $$DEV ]; then \
		sudo udevadm trigger --action=remove $$DEV ; \
	fi
	@sleep 3.0s
	@uhubctl -l 1-2.4 -a on -p 1

powercycle-maixbit:
	@uhubctl -l 1-2 -a cycle -p 1
