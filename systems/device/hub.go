// Package device contains implementation of a device plugin wrappers.
package device

import (
	"github.com/pkg/errors"
	"go-home.io/x/server/plugins/common"
	"go-home.io/x/server/plugins/device"
	"go-home.io/x/server/plugins/device/enums"
	"go-home.io/x/server/providers"
	"go-home.io/x/server/systems"
	"go-home.io/x/server/systems/logger"
)

// Loads hub device.
// Hub is different from other devices, since it can operate multiple different devices.
// nolint: dupl
func loadHub(ctor *ConstructDevice, pluginLogger common.IPluginLoggerProvider) ([]IDeviceWrapperProvider, error) {
	wrappers := make([]IDeviceWrapperProvider, 0)

	loadData := &device.InitDataDevice{
		Logger:                pluginLogger,
		Secret:                ctor.Settings.Secrets(),
		UOM:                   ctor.UOM,
		DeviceDiscoveredChan:  make(chan *device.DiscoveredDevices, 3),
		DeviceStateUpdateChan: make(chan *device.StateUpdateData, 10),
	}

	pluginLoadRequest := &providers.PluginLoadRequest{
		InitData:       loadData,
		RawConfig:      []byte(ctor.RawConfig),
		PluginProvider: ctor.DeviceName,
		SystemType:     systems.SysDevice,
		ExpectedType:   device.TypeHub,
	}
	i, err := ctor.Settings.PluginLoader().LoadPlugin(pluginLoadRequest)
	if err != nil {
		pluginLogger.Error("Failed to load hub plugin", err)
		return nil, errors.Wrap(err, "plugin load failed")
	}

	hub := i.(device.IHub)

	hubResults, err := hub.Load()
	if err != nil {
		pluginLogger.Error("Failed to load hub devices", err)
		return nil, errors.Wrap(err, "plugin init failed")
	}

	hubCtor := &wrapperConstruct{
		DeviceType:        enums.DevHub,
		DeviceInterface:   hub,
		IsRootDevice:      true,
		DeviceConfigName:  ctor.ConfigName,
		DeviceProvider:    ctor.DeviceName,
		DeviceState:       hubResults.State,
		LoadData:          loadData,
		Logger:            pluginLogger,
		SystemLogger:      ctor.Settings.PluginLogger(),
		Secret:            ctor.Settings.Secrets(),
		WorkerID:          ctor.Settings.NodeID(),
		Validator:         ctor.Settings.Validator(),
		DiscoveryChan:     ctor.DiscoveryChan,
		StatusUpdatesChan: ctor.StatusUpdatesChan,
		UOM:               ctor.UOM,
		processor:         nil,
		RawConfig:         ctor.RawConfig,
	}

	hubWrapper := NewDeviceWrapper(hubCtor)
	wrappers = append(wrappers, hubWrapper)

	for _, v := range hubResults.Devices {
		subLoadData := &device.InitDataDevice{
			Logger:                pluginLogger,
			Secret:                ctor.Settings.Secrets(),
			UOM:                   ctor.UOM,
			DeviceDiscoveredChan:  loadData.DeviceDiscoveredChan,
			DeviceStateUpdateChan: make(chan *device.StateUpdateData, 10),
		}

		dev, ok := v.Interface.(device.IDevice)
		if !ok {
			pluginLogger.Warn("One of the loaded devices is not implementing IDevice interface")
			continue
		}

		err := dev.Init(subLoadData)
		if err != nil {
			pluginLogger.Error("Failed to load hub device", err)
			continue
		}

		logCtor := &logger.ConstructPluginLogger{
			SystemLogger: ctor.Settings.PluginLogger(),
			Provider:     ctor.DeviceName,
			System:       systems.SysDevice.String(),
			ExtraFields: map[string]string{
				common.LogNameToken:       ctor.ConfigName,
				common.LogDeviceTypeToken: v.Type.String(),
			},
		}

		log := logger.NewPluginLogger(logCtor)

		spawnedCtor := &wrapperConstruct{
			DeviceType:        v.Type,
			DeviceInterface:   v.Interface,
			IsRootDevice:      false,
			DeviceConfigName:  ctor.ConfigName,
			DeviceProvider:    ctor.DeviceName,
			DeviceState:       v.State,
			LoadData:          subLoadData,
			Logger:            log,
			SystemLogger:      ctor.Settings.PluginLogger(),
			Secret:            ctor.Settings.Secrets(),
			WorkerID:          ctor.Settings.NodeID(),
			Validator:         ctor.Settings.Validator(),
			DiscoveryChan:     ctor.DiscoveryChan,
			StatusUpdatesChan: ctor.StatusUpdatesChan,
			UOM:               ctor.UOM,
			processor:         newDeviceProcessor(v.Type, ctor.RawConfig),
			RawConfig:         ctor.RawConfig,
		}

		w := NewDeviceWrapper(spawnedCtor)
		if nil == w {
			continue
		}
		wrappers = append(wrappers, w)
	}

	return wrappers, nil
}
