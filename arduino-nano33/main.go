package main

// Integration tests for Arduino Nano33 IoT
//
// Wire up the pins, and run it while connected to the USB port.
//
// Digital read/write tests:
//	D12 <--> G
//	D11 <--> 3V
//	D10 <--> D9
//
// Analog read tests:
//	A0 <--> 3.3V
//	A1 <--> 3.3V/2 (use voltage divider)
//	A2 <--> G
//
// I2C tests:
// 	Arduino A5 <--> MPU-6050 SCL
// 	Arduino A4 <--> MPU-6050 SDA
// 	Arduino G <--> MPU-6050 GND
// 	Arduino D7 <--> MPU-6050 VCC
//
import (
	"machine"

	"time"

	"tinygo.org/x/drivers/mpu6050"
)

var (
	// used by digital tests
	readV    = machine.D11
	readG    = machine.D12
	readpin  = machine.D9
	writepin = machine.D10

	// used by analog tests
	analogV    = machine.ADC{machine.A0}
	analogHalf = machine.ADC{machine.A1}
	analogG    = machine.ADC{machine.A2}

	// used by i2c tests
	accel    *mpu6050.Device
	powerpin = machine.D7

	serial = machine.UART0
)

const (
	maxanalog         = 65535
	allowedvariance   = 1024
	numberAnalogReads = 10
)

func main() {
	serial.Configure(machine.UARTConfig{})
	machine.I2C0.Configure(machine.I2CConfig{})
	machine.InitADC()

	waitForStart()

	digitalReadVoltage()
	digitalReadGround()
	digitalWrite()
	analogReadVoltage()
	analogReadGround()
	analogReadHalfVoltage()
	i2cConnection()

	endTests()
}

// wait for keypress on serial port to start test suite.
func waitForStart() {
	time.Sleep(3 * time.Second)

	println("=== TINYGO INTEGRATION TESTS ===")
	println("Press 't' key to begin running tests...")

	for {
		if serial.Buffered() > 0 {
			data, _ := serial.ReadByte()

			if data != 't' {
				time.Sleep(100 * time.Millisecond)
			}
			return
		}
	}
}

func endTests() {
	println("\n### Tests complete.")

	// tests done, now sleep waiting for baud reset to load new code
	for {
		time.Sleep(1 * time.Second)
	}
}

// digital read of D11 pin physically connected to V
func digitalReadVoltage() {
	printtest("digitalReadVoltage")

	readV.Configure(machine.PinConfig{Mode: machine.PinInput})

	// should be on
	if readV.Get() {
		printtestresult("pass")
		return
	}

	printtestresult("fail")
}

// digital read of D12 pin physically connected to G
func digitalReadGround() {
	printtest("digitalReadGround")

	readG.Configure(machine.PinConfig{Mode: machine.PinInput})

	// should be off
	if readG.Get() {
		printtestresult("fail")
		return
	}

	printtestresult("pass")
}

// digital write on/off of D9 pin as input physically connected to D10 pin as output.
func digitalWrite() {
	readpin.Configure(machine.PinConfig{Mode: machine.PinInput})
	writepin.Configure(machine.PinConfig{Mode: machine.PinOutput})

	printtest("digitalWriteOn")
	writepin.High()
	time.Sleep(100 * time.Millisecond)

	// should be on
	if readpin.Get() {
		printtestresult("pass")
	} else {
		printtestresult("fail")
	}

	time.Sleep(100 * time.Millisecond)

	printtest("digitalWriteOff")
	writepin.Low()
	time.Sleep(100 * time.Millisecond)

	// should be off
	if readpin.Get() {
		printtestresult("fail")
		return
	} else {
		printtestresult("pass")
	}
}

// analog read of pin connected to supply voltage.
func analogReadVoltage() {
	analogV.Configure()

	printtest("analogReadVoltage")

	// should be close to max
	var avg int
	for i := 0; i < numberAnalogReads; i++ {
		v := analogV.Get()
		avg += int(v)
		time.Sleep(100 * time.Millisecond)
	}
	avg /= numberAnalogReads
	val := uint16(avg)

	if val >= maxanalog-allowedvariance {
		printtestresult("pass")

		return
	} else {
		printtestresult("fail")
		printfailexpected("'val >= 65535-1024'")
		printfailactual(val)
	}
}

// analog read of pin connected to ground.
func analogReadGround() {
	analogG.Configure()

	printtest("analogReadGround")

	// should be close to zero
	var avg int
	for i := 0; i < numberAnalogReads; i++ {
		v := analogG.Get()
		avg += int(v)
		time.Sleep(100 * time.Millisecond)
	}
	avg /= numberAnalogReads
	val := uint16(avg)

	if val <= allowedvariance {
		printtestresult("pass")
		return
	} else {
		printtestresult("fail")

		printfailexpected("'val <= 1024'")
		printfailactual(val)
	}
}

// analog read of pin connected to supply voltage that has been divided by 2
// using resistors.
func analogReadHalfVoltage() {
	analogHalf.Configure()

	printtest("analogReadHalfVoltage")

	// should be around half the max
	var avg int
	for i := 0; i < numberAnalogReads; i++ {
		v := analogHalf.Get()
		avg += int(v)
		time.Sleep(100 * time.Millisecond)
	}
	avg /= numberAnalogReads
	val := uint16(avg)

	if val <= maxanalog/2+allowedvariance && val >= maxanalog/2-allowedvariance {
		printtestresult("pass")
		return
	}
	printtestresult("fail")
	printfailexpected("'val <= 65535/2+1024 && val >= 65535/2-1024'")
	printfailactual(val)
}

// checks to see if an attached MPU-6050 accelerometer is connected.
func i2cConnection() {
	powerpin.Configure(machine.PinConfig{Mode: machine.PinOutput})

	a := mpu6050.New(machine.I2C0)
	accel = &a

	printtest("i2cConnectionNoPower")

	// should not be connected when not powered
	powerpin.Low()
	time.Sleep(100 * time.Millisecond)
	if accel.Connected() {
		printtestresult("fail")
	} else {
		printtestresult("pass")
	}

	printtest("i2cConnectionPower")
	// turn on power and should be connected now
	powerpin.High()
	time.Sleep(100 * time.Millisecond)

	accel.Configure()
	time.Sleep(400 * time.Millisecond)

	if !accel.Connected() {
		printtestresult("fail")
		return
	}

	printtestresult("pass")
}

func printtest(testname string) {
	print("- " + testname + " = ")
}

func printtestresult(result string) {
	println("***" + result + "***")
}

func printfailexpected(reason string) {
	println("        expected:", reason)
}

func printfailactual(val uint16) {
	println("        actual:", val)
}
