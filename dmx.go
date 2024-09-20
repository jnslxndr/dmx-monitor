package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go.bug.st/serial/enumerator"

	"github.com/H3rby7/usbdmx-golang/controller/enttec/dmxusbpro"
	"github.com/tarm/serial"
)

const MAX_CHANNELS = 512

type DeviceInfo struct {
	Path         string
	Name         string
	SerialNumber string
}

type DeviceInfos []DeviceInfo

func (d DeviceInfo) String() string {
	return d.Path + " " + d.SerialNumber
}

func (ip DeviceInfos) String() string {
	var bf strings.Builder

	for i, p := range ip {
		bf.WriteString(fmt.Sprintf("[%v] %s\n", i, p))
	}

	return bf.String()
}

func Observer(found func(DeviceInfos)) {
	go func() {
		list, err := ListDMXDevices()
		if err == nil {
			found(list)
		} else {
			found(DeviceInfos{})
		}
		time.AfterFunc(time.Millisecond*1000, func() { Observer(found) })
	}()
}

func ListDMXDevices() (DeviceInfos, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, err
	}

	list := make([]DeviceInfo, 0)

	for _, p := range ports {
		if isEnttec(p) {
			list = append(list, DeviceInfo{p.Name, p.Product, p.SerialNumber})
		}
	}
	if len(list) == 0 {
		return list, errors.New("no device found")
	}

	return list, nil
}

func FindDevice(path string) (*DeviceInfo, error) {
	devices, err := ListDMXDevices()
	if err != nil {
		return nil, err
	}
	for _, d := range devices {
		if d.Path == path {
			return &d, nil
		}
	}
	return nil, errors.New("not found")
}

func FirstDevice() (*DeviceInfo, error) {
	devices, err := ListDMXDevices()
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, errors.New("not found")
	}
	return &devices[0], nil
}

func isEnttec(p *enumerator.PortDetails) bool {
	// 		manufacturer == 'ENTTEC' ||
	// 		vendorId == '0403' && productId == '6001'
	return p.VID == "0403" && p.PID == "6001"
}

func NewConnection(info *DeviceInfo) (*dmxusbpro.EnttecDMXUSBProController, error) {
	return newConnection(info, true)
}

func NewReadConnection(info *DeviceInfo) (*dmxusbpro.EnttecDMXUSBProController, error) {
	return newConnection(info, false)
}

func newConnection(info *DeviceInfo, write bool) (*dmxusbpro.EnttecDMXUSBProController, error) {
	config := &serial.Config{Name: info.Path, Baud: 57600}
	// FIXME Make max channels a clis arg and env var
	controller := dmxusbpro.NewEnttecDMXUSBProController(config, MAX_CHANNELS, write)
	if err := controller.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect DMX Controller: %s", err)
	}
	return controller, nil
}
