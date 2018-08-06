// Package server contains go-home server.
package server

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	busPlugin "github.com/go-home-io/server/plugins/bus"
	"github.com/go-home-io/server/plugins/common"
	"github.com/go-home-io/server/providers"
	"github.com/go-home-io/server/systems"
	"github.com/go-home-io/server/systems/api"
	"github.com/go-home-io/server/systems/bus"
	"github.com/go-home-io/server/systems/group"
	"github.com/go-home-io/server/systems/trigger"
	"github.com/gorilla/mux"
)

const (
	// Logger system representation.
	logSystem = "server"
)

// GoHomeServer describes master node.
type GoHomeServer struct {
	Settings      providers.ISettingsProvider
	Logger        common.ILoggerProvider
	MessageParser bus.IMasterMessageParserProvider

	incomingChan chan busPlugin.RawMessage

	state IServerStateProvider

	triggers     []providers.ITriggerProvider
	extendedAPIs []providers.IExtendedAPIProvider
	groups       map[string]providers.IGroupProvider
}

// NewServer constructs a new master server.
// nolint: dupl
func NewServer(settings providers.ISettingsProvider) (providers.IServerProvider, error) {
	server := GoHomeServer{
		Logger:        settings.SystemLogger(),
		Settings:      settings,
		MessageParser: bus.NewServerMessageParser(settings.SystemLogger()),

		incomingChan: make(chan busPlugin.RawMessage, 100),
	}

	server.state = newServerState(settings)

	return &server, nil
}

// Start launches master server.
func (s *GoHomeServer) Start() {
	prepareCidrs()

	s.startTriggers()
	s.startGroups()

	router := mux.NewRouter()
	s.registerAPI(router)
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", s.Settings.MasterSettings().Port), router)
		if err != nil {
			s.Logger.Fatal("Failed to start server", err, common.LogSystemToken, logSystem)
		}
	}()

	s.Logger.Info(fmt.Sprintf("Started server on port %d", s.Settings.MasterSettings().Port),
		common.LogSystemToken, logSystem)
	go func() {
		sl := s.Settings.MasterSettings().DelayedStart
		if sl > 0 {
			time.Sleep(time.Duration(sl) * time.Second)
		}

		s.busStart()
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	for range c {
		s.Logger.Info("Received stop command, exiting", common.LogSystemToken, logSystem)
		os.Exit(0)
	}
}

// GetDevice returns known device.
func (s *GoHomeServer) GetDevice(ID string) *providers.KnownDevice {
	kd := s.state.GetDevice(ID)
	if nil == kd {
		return nil
	}

	return &providers.KnownDevice{
		Commands: kd.Commands,
		Worker:   kd.Worker,
	}
}

// PushMasterDeviceUpdate pushed device to known devices state
func (s *GoHomeServer) PushMasterDeviceUpdate(update *providers.MasterDeviceUpdate) {
	msg := &bus.DeviceUpdateMessage{
		State:      update.State,
		Commands:   update.Commands,
		DeviceID:   update.ID,
		DeviceName: update.Name,
		DeviceType: update.Type,
		WorkerID:   "master",
	}

	s.state.Update(msg)
}

// All API registration.
func (s *GoHomeServer) registerAPI(router *mux.Router) {
	publicRouter := router.PathPrefix("/pub").Subrouter()
	publicRouter.HandleFunc("/ping", s.ping).Methods(http.MethodGet)

	apiRouter := router.PathPrefix(routeAPI).Subrouter()
	apiRouter.HandleFunc("/device", s.getDevices).Methods(http.MethodGet)
	apiRouter.HandleFunc(fmt.Sprintf("/device/{%s}/{%s}", urlDeviceID, urlCommandName),
		s.deviceCommand).Methods(http.MethodPost)
	apiRouter.HandleFunc("/group", s.getGroups).Methods(http.MethodGet)

	apiRouter.Use(s.logMiddleware)
	router.Use(s.authMiddleware)

	s.startAPI(router, apiRouter)
}

// Starting bus communications.
func (s *GoHomeServer) busStart() {
	err := s.Settings.ServiceBus().Subscribe(busPlugin.ChDiscovery, s.incomingChan)
	if err != nil {
		s.Logger.Fatal("Failed to subscribe to discovery channel", err, common.LogSystemToken, logSystem)
	}

	err = s.Settings.ServiceBus().Subscribe(busPlugin.ChDeviceUpdates, s.incomingChan)

	if err != nil {
		s.Logger.Fatal("Failed to subscribe to updates channel", err, common.LogSystemToken, logSystem)
	}

	s.Logger.Debug("Successfully subscribed to bus channels", common.LogSystemToken, logSystem)
	s.busCycle()
}

// Internal bus cycle.
func (s *GoHomeServer) busCycle() {
	for {
		select {
		case msg := <-s.incomingChan:
			go s.MessageParser.ProcessIncomingMessage(&msg)
		case dis := <-s.MessageParser.GetDiscoveryMessageChan():
			s.state.Discovery(dis)
		case dup := <-s.MessageParser.GetDeviceUpdateMessageChan():
			s.state.Update(dup)
		}
	}
}

// Starts triggers
func (s *GoHomeServer) startTriggers() {
	s.triggers = make([]providers.ITriggerProvider, 0)
	for _, v := range s.Settings.Triggers() {
		ctor := &trigger.ConstructTrigger{
			Logger:    s.Settings.PluginLogger(systems.SysTrigger, v.Provider),
			Provider:  v.Provider,
			RawConfig: v.RawConfig,
			Loader:    s.Settings.PluginLoader(),
			FanOut:    s.Settings.FanOut(),
			Secret:    s.Settings.Secrets(),
			Validator: s.Settings.Validator(),
			Server:    s,
		}
		tr, err := trigger.NewTrigger(ctor)
		if err != nil {
			continue
		}

		s.triggers = append(s.triggers, tr)
	}
}

// Starts APIs
func (s *GoHomeServer) startAPI(root *mux.Router, external *mux.Router) {
	s.extendedAPIs = make([]providers.IExtendedAPIProvider, 0)
	for _, v := range s.Settings.ExtendedAPIs() {
		ctor := &api.ConstructAPI{
			Provider:           v.Provider,
			RawConfig:          v.RawConfig,
			Server:             s,
			Validator:          s.Settings.Validator(),
			Logger:             s.Settings.PluginLogger(systems.SysAPI, v.Provider),
			Secret:             s.Settings.Secrets(),
			Loader:             s.Settings.PluginLoader(),
			FanOut:             s.Settings.FanOut(),
			InternalRootRouter: root,
			ExternalAPIRouter:  external,
			IsServer:           true,
			ServiceBus:         s.Settings.ServiceBus(),
			Name:               v.Name,
		}

		a, err := api.NewExtendedAPIProvider(ctor)
		if err != nil {
			continue
		}

		s.extendedAPIs = append(s.extendedAPIs, a)
	}
}

// Starts groups
func (s *GoHomeServer) startGroups() {
	s.groups = make(map[string]providers.IGroupProvider)

	for _, v := range s.Settings.Groups() {
		ctor := &group.ConstructGroup{
			RawConfig: v.RawConfig,
			Settings:  s.Settings,
			Server:    s,
		}

		g, err := group.NewGroupProvider(ctor)
		if err != nil {
			continue
		}

		s.groups[g.ID()] = g
	}
}
