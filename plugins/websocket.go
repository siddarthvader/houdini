package plugins

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path"
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

	if _, ok := plugin.(GenerateRuntime); ok {
		hooks["GenerateRuntime"] = true
		registerWebsocketGenerateHandler(plugin)
	}
	// generate endpoint for websocket
	if _, ok := plugin.(GenerateDocuments); ok {
		hooks["GenerateDocuments"] = true
		registerWebsocketDocumentsHandler(plugin)
	}

	if _, ok := plugin.(Schema); ok {
		hooks["Schema"] = true
		registerWebsocketSchemaHandler(plugin)
	}

	if _, ok := plugin.(AfterExtract); ok {
		hooks["AfterExtract"] = true
		registerWebsocketAfterExtractHandler(plugin)
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

	// message loop
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

	wsHandlers["GenerateRuntime"] = func(conn *websocket.Conn, msg WebSocketMessage) {
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
			log.Printf(" taskId is empty for GenerateDocuments hook")
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

func registerWebsocketDocumentsHandler(plugin HoudiniPlugin[config.PluginConfig]) {
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
			log.Printf(" taskId is empty for GenerateDocuments hook")
		}
		// put the wsconn & msgId into the context
		ctx := ContextWithWSConn(context.Background(), conn)
		log.Printf("WSMessageID: %s", msg.ID)
		ctx = ContextWithWSMessageID(ctx, msg.ID)
		ctx = ContextWithPluginDir(
			ContextWithTaskID(ctx, taskID),
			msg.PluginDirectory,
		)
		result, err := handleGenerateDocuments(plugin)(ctx)
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

func registerWebsocketSchemaHandler(plugin HoudiniPlugin[config.PluginConfig]) {
	wsMutex.Lock()
	defer wsMutex.Unlock()
	wsHandlers["Schema"] = func(conn *websocket.Conn, msg WebSocketMessage) {
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
			log.Printf(" taskId is empty for Schema hook")
		}
		// put the wsconn & msgId into the context
		ctx := ContextWithWSConn(context.Background(), conn)
		log.Printf("WSMessageID: %s", msg.ID)
		ctx = ContextWithWSMessageID(ctx, msg.ID)
		ctx = ContextWithPluginDir(
			ContextWithTaskID(ctx, taskID),
			msg.PluginDirectory,
		)
		err := handleSchema(plugin)(ctx)
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
			Result: nil,
		}
		if writeErr := conn.WriteJSON(response); writeErr != nil {
			log.Printf("Failed to write response: %s", writeErr.Error())
		}
	}
}

func handleWebsocketHookHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received HTTP POST request to /ws endpoint")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","message":"HTTP fallback for WebSocket endpoint"}`))
}

// generator plugin functions
func handleGenerateRuntime[PluginConfig any](
	plugin HoudiniPlugin[PluginConfig],
) func(ctx context.Context) ([]string, error) {
	return func(ctx context.Context) ([]string, error) {
		// if conn := WSConnFromContext(ctx); conn != nil {
		// 	conn.WriteJSON(WebSocketResponse{
		// 		ID:    ctx.Value("wsMessageID").(string),
		// 		Type:  "1error",
		// 		Error: "TEST: This is a simulated non-fatal error during generation",
		// 	})
		// }
		paths := []string{}

		if generate, ok := plugin.(GenerateRuntime); ok {
			filepaths, err := generate.GenerateRuntime(ctx)
			if err != nil {
				return nil, err
			}

			paths = append(paths, filepaths...)
		}

		// if the plugin defines a runtime to be included then we should include it now
		if includeRuntime, ok := plugin.(IncludeRuntime); ok {
			runtimeDir, err := includeRuntime.IncludeRuntime(ctx)
			if err != nil {
				return nil, err
			}

			config, err := plugin.Database().ProjectConfig(ctx)
			if err != nil {
				return nil, err
			}

			runtimePath := path.Join(PluginDirFromContext(ctx), runtimeDir)
			targetPath := config.PluginRuntimeDirectory(plugin.Name())

			// the plugin could have defined a transform for the runtime
			transform := func(ctx context.Context, source string, content string) (string, error) { return content, nil }
			if transformer, ok := plugin.(TransformRuntime); ok {
				transform = transformer.TransformRuntime
			}

			// copy the plugin runtime to the runtime directory
			updated, err := RecursiveCopy(ctx, runtimePath, targetPath, transform)
			if err != nil {
				return nil, err
			}

			// add any updated paths to the list
			paths = append(paths, updated...)
		}

		// nothing went wrong
		return paths, nil
	}
}

func handleGenerateDocuments[PluginConfig any](
	plugin HoudiniPlugin[PluginConfig],
) func(ctx context.Context) (any, error) {
	return func(ctx context.Context) (any, error) {
		if generate, ok := plugin.(GenerateDocuments); ok {
			return generate.GenerateDocuments(ctx)
		}
		return nil, nil
	}
}

func handleSchema[PluginConfig any](
	plugin HoudiniPlugin[PluginConfig],
) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if schema, ok := plugin.(Schema); ok {
			return schema.Schema(ctx)
		}
		return fmt.Errorf("schema hook not implemented")
	}
}

func registerWebsocketAfterExtractHandler(plugin HoudiniPlugin[config.PluginConfig]) {
	wsMutex.Lock()
	defer wsMutex.Unlock()
	wsHandlers["AfterExtract"] = func(conn *websocket.Conn, msg WebSocketMessage) {
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
			log.Printf(" taskId is empty for AfterExtract hook")
		}
		// put the wsconn & msgId into the context
		ctx := ContextWithWSConn(context.Background(), conn)
		log.Printf("WSMessageID: %s", msg.ID)
		ctx = ContextWithWSMessageID(ctx, msg.ID)
		ctx = ContextWithPluginDir(
			ContextWithTaskID(ctx, taskID),
			msg.PluginDirectory,
		)
		err := handleAfterExtract(plugin)(ctx)
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
			Result: nil,
		}
		if writeErr := conn.WriteJSON(response); writeErr != nil {
			log.Printf("Failed to write response: %s", writeErr.Error())
		}
	}
}

func handleAfterExtract[PluginConfig any](
	plugin HoudiniPlugin[PluginConfig],
) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if afterExtract, ok := plugin.(AfterExtract); ok {
			return afterExtract.AfterExtract(ctx)
		}
		return fmt.Errorf("afterExtract hook not implemented")
	}
}
