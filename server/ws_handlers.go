package server

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
	"go-home.io/x/server/plugins/common"
	"go-home.io/x/server/providers"
)

type wsCmd struct {
	ID  string      `json:"id"`
	Cmd string      `json:"cmd"`
	Val interface{} `json:"value"`
}

// Handles WS upgrade request.
func (s *GoHomeServer) handleWS(writer http.ResponseWriter, request *http.Request) {
	usr := getContextUser(request)
	c, err := s.wsSettings.Upgrade(writer, request, nil)
	if err != nil {
		s.Logger.Error("Failed to establish a WS connection", err, common.LogUserNameToken, usr.Name())
		return
	}

	go s.processWSConnection(c, usr)
}

// Processes incoming WS connections.
//noinspection GoUnhandledErrorResult
func (s *GoHomeServer) processWSConnection(conn *websocket.Conn, usr providers.IAuthenticatedUser) {
	stop := make(chan bool, 1)
	go s.processIncomingWSMessages(conn, stop, usr)
	subID, upd := s.Settings.FanOut().SubscribeDeviceUpdates()
	defer s.Settings.FanOut().UnSubscribeDeviceUpdates(subID)

	for {
		select {
		case msg := <-stop:
			if msg {
				return
			}
		case msg, ok := <-upd:
			{
				if !ok {
					return
				}

				kd := s.state.GetDevice(msg.ID)
				if usr.DeviceGet(kd.ID) {
					conn.WriteJSON(kd) // nolint: gosec
				}
			}
		}
	}
}

// Processes incoming WS messages.
//noinspection GoUnhandledErrorResult
func (s *GoHomeServer) processIncomingWSMessages(conn *websocket.Conn, stop chan bool,
	usr providers.IAuthenticatedUser) {
	defer conn.Close()
	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			s.Logger.Info("Closing WS connection for user", common.LogUserNameToken, usr.Name())
			stop <- true
			return
		}

		// Ping request comes as a un-wrapped string
		if 4 == len(message) {
			conn.WriteMessage(mt, []byte("pong")) // nolint: gosec
			continue
		}

		cmd := &wsCmd{}
		err = json.Unmarshal(message, cmd)
		if err != nil {
			s.Logger.Error("Failed to un-marshal WS command", err, common.LogUserNameToken, usr.Name())
			continue
		}

		data, err := json.Marshal(cmd.Val)
		if err != nil {
			s.Logger.Error("Failed to marshal WS command", err, common.LogUserNameToken, usr.Name())
			continue
		}

		s.commandInvokeDeviceCommand(usr, cmd.ID, cmd.Cmd, data) // nolint: gosec
	}
}
