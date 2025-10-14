package plugins

import (
	"context"
	"fmt"
	"log"
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

func pluginWebsocketHooks(ctx context.Context, plugin HoudiniPlugin[config.PluginConfig]) []string {
	hooks := []string{}

	if _, ok := plugin.(GenerateRuntime); ok {
		hooks = append(hooks, "GenerateRuntime")
		registerWSHandler("GenerateRuntime", handleGenerateRuntime(plugin))
	}

	if _, ok := plugin.(GenerateDocuments); ok {
		hooks = append(hooks, "GenerateDocuments")
		registerWSHandler("GenerateDocuments", handleGenerateDocuments(plugin))
	}

	if _, ok := plugin.(Schema); ok {
		hooks = append(hooks, "Schema")
		registerWSHandler("Schema", handleSchema(plugin))
	}

	if _, ok := plugin.(AfterExtract); ok {
		hooks = append(hooks, "AfterExtract")
		registerWSHandler("AfterExtract", handleAfterExtract(plugin))
	}

	return hooks
}

// Register a WebSocket handler
func registerWSHandler[T any](hookName string, handler func(ctx context.Context) (T, error)) {
	wsMutex.Lock()
	defer wsMutex.Unlock()
	wsHandlers[hookName] = createWSHandler(hookName, handler)
}

// handler wrapper that handles common request/response logic
func createWSHandler[T any](
	hookName string,
	handler func(ctx context.Context) (T, error),
) func(*websocket.Conn, WebSocketMessage) {
	return func(conn *websocket.Conn, msg WebSocketMessage) {
		// validate request type
		if msg.Type != "request" {
			sendErrorResponse(conn, msg.ID, "Expected request type")
			return
		}

		if msg.TaskID == "" {
			log.Printf(" taskId is empty for %s hook", hookName)
		}

		// context with all necessary values
		ctx := ContextWithWSConn(context.Background(), conn)
		ctx = ContextWithWSMessageID(ctx, msg.ID)
		ctx = ContextWithTaskID(ctx, msg.TaskID)
		ctx = ContextWithPluginDir(ctx, msg.PluginDirectory)

		log.Printf("WSMessageID: %s", msg.ID)

		// execute
		result, err := handler(ctx)
		if err != nil {
			sendErrorResponse(conn, msg.ID, err.Error())
			return
		}

		// success response
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
) func(ctx context.Context) (any, error) {
	return func(ctx context.Context) (any, error) {
		if schema, ok := plugin.(Schema); ok {
			return nil, schema.Schema(ctx)
		}
		return nil, fmt.Errorf("schema hook not implemented")
	}
}

func handleAfterExtract[PluginConfig any](
	plugin HoudiniPlugin[PluginConfig],
) func(ctx context.Context) (any, error) {
	return func(ctx context.Context) (any, error) {
		if afterExtract, ok := plugin.(AfterExtract); ok {
			return nil, afterExtract.AfterExtract(ctx)
		}
		return nil, fmt.Errorf("afterExtract hook not implemented")
	}
}

// ws connection handler utils
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
