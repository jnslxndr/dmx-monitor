package main

import (
	"fmt"
	"os"
	"time"

	"github.com/H3rby7/usbdmx-golang/controller/enttec/dmxusbpro"
	"github.com/H3rby7/usbdmx-golang/controller/enttec/dmxusbpro/messages"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	Observer(func(di DeviceInfos) {
		p.Send(di)
	})
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

type model struct {
	devices     DeviceInfos
	controller  *dmxusbpro.EnttecDMXUSBProController
	showMonitor bool
	changes     chan messages.EnttecDMXUSBProApplicationMessage
	channels    []byte
}

func initialModel() model {
	return model{
		changes:  make(chan messages.EnttecDMXUSBProApplicationMessage),
		channels: make([]byte, MAX_CHANNELS),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

type dmxMonitor int

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case dmxMonitor:
		if msg == 1 {
			m.showMonitor = true
			select {
			case <-time.After(time.Millisecond * 20):
				// body = ""
				break
			case msg := <-m.changes:
				cs, err := messages.ToChangeSet(msg)
				if err == nil {
					for channel, value := range cs {
						if channel >= MAX_CHANNELS {
							continue
						}
						m.channels[channel] = value
					}
				}
			}
			return m, tea.Every(time.Millisecond*40, func(t time.Time) tea.Msg { return dmxMonitor(1) })
		} else {
			m.showMonitor = false
			return m, tea.ClearScreen
		}
	case DeviceInfos:
		m.devices = msg
		if len(m.devices) > 0 {
			if m.controller != nil {
				return m, nil
			}
			// Otherwise setup a new connection
			// clear the previous channel buffer
			m.channels = make([]byte, MAX_CHANNELS)
			var err error
			m.controller, err = NewReadConnection(&msg[0])
			if err != nil {
				return m, tea.ClearScreen
			}
			err = m.controller.SwitchReadMode(1)
			if err != nil {
				return m, tea.ClearScreen
			}
			go func() {
				defer func() {
					recover()
					m.controller.Disconnect()
					m.controller = nil
					m.showMonitor = false
				}()
				m.controller.OnDMXChange(m.changes, 40)
			}()
			return m, tea.Every(time.Millisecond*40, func(t time.Time) tea.Msg { return dmxMonitor(1) })
		} else {
			if m.controller != nil {
				m.controller.Disconnect()
				m.controller = nil
				m.showMonitor = false
			}
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.controller == nil {
		return fmt.Sprintf(
			"*** DMX Monitor *** %s\r\nNo DMX USB Pro connected...",
			time.Now().Format("15:04:05"),
		)
	}
	header := "Found the following devices\n"
	body := m.devices.String()

	if m.controller != nil && m.showMonitor {
		header = fmt.Sprintf(
			"*** DMX Monitor for %s *** %s\r\n",
			m.devices[0].Path,
			time.Now().Format("15:04:05.999999"),
		)
		body = channelView(m.channels, MAX_CHANNELS)
	}
	return header + body
}

func channelView(channels []byte, maxchannels int) string {
	cols := 16
	rows := maxchannels / cols
	view := []byte{}
	for y := 0; y < rows; y += 1 {
		view = fmt.Appendf(view, "Ch %03d - %03d >>> ", y*cols+1, y*cols+cols)
		for x := 0; x < cols; x++ {
			i := y*cols + x
			end := " "
			if x == cols-1 {
				end = ""
			}
			view = fmt.Appendf(view, "%3d%s", channels[i], end)
		}
		view = fmt.Append(view, "\r\n")
	}
	return string(view)
}
