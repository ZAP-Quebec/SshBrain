package web

import (
	"io"
)

type Device interface {
	Id() string
	Name() string
}

type ConnectedDevice interface {
	Device
	OpenShell() (stdin io.Writer, stdout io.Reader, stderr io.Reader, err error)
	Execute(command string) (stdout io.Reader, stderr io.Reader, exitCode <-chan int, err error)
}

type DeviceManager interface {
	GetConnectedDevicesAndEvents() (devices []ConnectedDevice, connectedDevices <-chan ConnectedDevice, disconnectedDevices <-chan Device)
}
