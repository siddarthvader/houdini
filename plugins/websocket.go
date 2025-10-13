package plugins

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"code.houdinigraphql.com/packages/houdini-core/config"
	"github.com/gorilla/websocket"
)

type WebSocketMessage struct {
	ID              string         `json:"id"`
	Type            string         `json:"type"`
	Hook            string         `json:"hook"`
	Payload         map[string]any `json:"payload"`
	TaskID          string         `json:"taskId"`
	PluginDirectory string         `json:"pluginDirectory"`
}

type WebSocketResponse struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// routing map for websocket handlers
var (
	wsHandlers = make(map[string]func(*websocket.Conn, WebSocketMessage))
	wsMutex    = sync.Mutex{}
)

func pluginWebsocketHooks(ctx context.Context, plugin HoudiniPlugin[config.PluginConfig]) error {
	hooks := map[string]bool{}

	// generate endpoint for websocket

	if _, ok := plugin.(GenerateDocuments); ok {
		hooks["GenerateDocuments"] = true
		registerWebsocketGenerateHandler(plugin)
	}

	if p, ok := plugin.(Schema); ok {
		hooks["Schema"] = true
		registerWebsocketSchemaHandler(p)
	}

	return nil
}

func HandleWebSocketConnection(conn *websocket.Conn) {
	log.Printf("WebSocket connection established from %s", conn.RemoteAddr())
	defer log.Printf("WebSocket connection closed from %s", conn.RemoteAddr())

	// ping
	conn.SetPingHandler(func(string) error {
		// pong
		return conn.WriteMessage(websocket.PongMessage, []byte{})
	})

	conn.SetReadDeadline(time.Now().Add(time.Second * 60))

	// ,essage loop
	for {
		var msg WebSocketMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.
				CloseAbnormalClosure) {
				log.Printf("WebSocket connection closed normally from %s: %v", conn.RemoteAddr(),
					err)
			} else {
				log.Printf("WebSocket read error from %s: %v", conn.RemoteAddr(), err)
			}
			break
		}

		// can read Now, reset read deadling
		conn.SetReadDeadline(time.Now().Add(time.Second * 60))

		log.Printf("Received messaes from %s that is %s, %v", conn.RemoteAddr(), msg.Type, msg)

		wsMutex.Lock()
		handler, exists := wsHandlers[msg.Hook]
		wsMutex.Unlock()

		if exists {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Handler panic for hook %s: %v", msg.Hook, r)
						sendErrorResponse(conn, msg.ID, fmt.Sprintf("Handler panic: %v", r))
					}
				}()

				handler(conn, msg)
			}()
		} else {
			log.Printf("No handler for hook %s", msg.Hook)
			sendErrorResponse(conn, msg.ID, fmt.Sprintf("No handler for hook %s", msg.Hook))
		}
	}
}

func sendErrorResponse(conn *websocket.Conn, id string, err string) {
	response := WebSocketResponse{
		ID:    id,
		Type:  "response",
		Error: err,
	}
	if writeErr := conn.WriteJSON(response); writeErr != nil {
		log.Printf("Failed to write response: %s", writeErr.Error())
	}
}

func registerWebsocketGenerateHandler(plugin HoudiniPlugin[config.PluginConfig]) {
	wsMutex.Lock()
	defer wsMutex.Unlock()

	wsHandlers["GenerateDocuments"] = func(conn *websocket.Conn, msg WebSocketMessage) {
		if msg.Type != "request" {
			response := WebSocketResponse{
				ID:    msg.ID,
				Type:  "response",
				Error: "Expected request type",
			}
			conn.WriteJSON(response)
			return
		}

		// Handle empty or missing taskId
		taskID := msg.TaskID
		if taskID == "" {
			log.Printf("Warning: taskId is empty for GenerateDocuments hook")
		}

		// put the wsconn & msgId into the context
		ctx := ContextWithWSConn(context.Background(), conn)
		log.Printf("WSMessageID: %s", msg.ID)
		ctx = ContextWithWSMessageID(ctx, msg.ID)

		// Set up context with taskID and plugin directory
		ctx = ContextWithPluginDir(
			ContextWithTaskID(ctx, taskID),
			msg.PluginDirectory,
		)

		result, err := handleGenerateRuntime(plugin)(ctx)
		if err != nil {
			response := WebSocketResponse{
				ID:    msg.ID,
				Type:  "response",
				Error: err.Error(),
			}
			if writeErr := conn.WriteJSON(response); writeErr != nil {
				log.Printf("Failed to write response: %s", writeErr.Error())
			}
			return
		}

		response := WebSocketResponse{
			ID:     msg.ID,
			Type:   "response",
			Result: result,
		}
		if writeErr := conn.WriteJSON(response); writeErr != nil {
			log.Printf("Failed to write response: %s", writeErr.Error())
		}
	}
}
func registerWebsocketSchemaHandler(p Schema) {}

func handleWebsocketHookHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received HTTP POST request to /ws endpoint")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","message":"HTTP fallback for WebSocket endpoint"}`))
}
