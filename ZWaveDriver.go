package main

import (
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"

	"github.com/ninjasphere/go-openzwave"
	"github.com/ninjasphere/go-openzwave/NT"
)

const (
	driverName = "com.ninjablocks.zwave"
)

var (
	log  = logger.GetLogger(driverName)
	info = ninja.LoadModuleInfo("./package.json")
)

/*model.Module{
	ID:          "com.ninjablocks.zwave",
	Name:        "ZWave Driver",
	Version:     "0.2.0",
	Description: "Ninja Blocks ZWave driver",
	Author:      "Jon Seymour <jon@ninjablocks.com>",
	License:     "MIT",
}*/

type zwaveDriver struct {
	config    *ZWaveDriverConfig
	conn      *ninja.Connection
	debug     bool
	zwaveAPI  openzwave.API
	exit      chan int
	sendEvent func(event string, payload interface{}) error
}

type ZWaveDriverConfig struct {
}

func defaultConfig() *ZWaveDriverConfig {
	return &ZWaveDriverConfig{}
}

func (driver *zwaveDriver) GetOpenZWaveAPI() openzwave.API {
	return driver.zwaveAPI
}

func (driver *zwaveDriver) GetNinjaDriver() ninja.Driver {
	return driver
}

func (driver *zwaveDriver) GetNinjaConnection() *ninja.Connection {
	return driver.conn
}

func newZWaveDriver(debug bool) (*zwaveDriver, error) {

	conn, err := ninja.Connect(driverName)

	if err != nil {
		log.Fatalf("Failed to create %s driver: %s", driverName, err)
	}

	driver := &zwaveDriver{
		config:    defaultConfig(),
		conn:      conn,
		sendEvent: nil,
		debug:     debug,
		zwaveAPI:  nil,
		exit:      make(chan int, 0),
	}

	err = conn.ExportDriver(driver)

	if err != nil {
		log.Fatalf("Failed to export %s driver: %s", driverName, err)
	}

	return driver, nil
}

func (d *zwaveDriver) Start(config *ZWaveDriverConfig) error {
	log.Infof("Driver %s starting with config %v", driverName, config)

	d.config = config

	zwaveDeviceFactory := func(api openzwave.API, node openzwave.Node) openzwave.Device {
		d.zwaveAPI = api
		return GetLibrary().GetDeviceFactory(*node.GetProductId())(d, node)
	}

	shuttingDown := false

	notificationCallback := func(api openzwave.API, nt openzwave.Notification) {
		switch nt.GetNotificationType().Code {
		case NT.NODE_REMOVED:
			//
			// Currently the RPC layer prevents us releasing the resources associated
			// with removed nodes. If the nodes come back (when, say, the zwave controller
			// is re-inserted), we can't build new device  wrappers for them because the
			// devices are already registered with the RPC layer.
			//
			// We could fix the RPC layer or we could attempt to work around the
			// problems with the RPC layer by using "patch" proxies for each ninja device
			// that allows us to change the actual zwave device.
			//
			// For now, it is simpler if we simply restart the driver process in the event of node
			// removal. This also avoids potential race conditions between
			// event dispatch and freeing of the resources associated with the
			// removed node.
			//
			if !shuttingDown {
				shuttingDown = true
				api.Logger().Infof("ZWave driver shutdown in response to node removed event.")
				api.Shutdown(openzwave.EXIT_NODE_REMOVED)
			}
		default:

		}
	}

	configurator := openzwave.
		BuildAPI("/usr/local/etc/openzwave", ".", "").
		SetLogger(log).
		SetNotificationCallback(notificationCallback).
		SetDeviceFactory(zwaveDeviceFactory)

	if d.debug {
		callback := func(api openzwave.API, notification openzwave.Notification) {
			api.Logger().Infof("%v\n", notification)
			notificationCallback(api, notification)
		}

		configurator.SetNotificationCallback(callback)
	}

	go func() {
		// slightly racy - we would like a guarantee we have replied to Start
		// before we start generating advice about new nodes.
		d.exit <- configurator.Run()
	}()

	d.sendEvent("config", config)

	return nil
}

func (d *zwaveDriver) Stop() error {
	// TODO: propagate shutdown request to ZWave driver and let it take it down
	log.Infof("Stop received - shutting down")
	d.exit <- 0
	return nil
}

// wait until the drivers are ready for us to shutdown.
func (d *zwaveDriver) Wait() int {
	return <-d.exit
}

func (d *zwaveDriver) GetModuleInfo() *model.Module {
	return info
}

func (d *zwaveDriver) SetEventHandler(sendEvent func(event string, payload interface{}) error) {
	d.sendEvent = sendEvent
}