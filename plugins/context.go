package plugins

import (
	"context"
	"strconv"

	"github.com/gorilla/websocket"
)

func ContextWithTaskID(ctx context.Context, taskID string) context.Context {
	if taskID == "" {
		return ctx
	}

	id, err := strconv.ParseInt(taskID, 10, 64)
	if err != nil {
		return ctx
	}

	return context.WithValue(ctx, "taskID", &id)
}

func ContextWithPluginDir(ctx context.Context, directory string) context.Context {
	return context.WithValue(ctx, "pluginDir", directory)
}

func TaskIDFromContext(ctx context.Context) *int64 {
	taskID := ctx.Value("taskID")
	if taskID == nil {
		return nil
	}
	return taskID.(*int64)
}

func PluginDirFromContext(ctx context.Context) string {
	return ctx.Value("pluginDir").(string)
}

func ContextWithWSConn(ctx context.Context, conn *websocket.Conn) context.Context {
	return context.WithValue(ctx, "wsConn", conn)
}

// example of how to use the WSConnFromContext function
//if conn := WSConnFromContext(ctx); conn != nil {
// 	conn.WriteJSON(WebSocketResponse{
// 		ID:    ctx.Value("wsMessageID").(string),
// 		Type:  "1error",
// 		Error: "TEST: This is a simulated non-fatal error during generation",
// 	})
// }
// helper to get the conn from context,
func WSConnFromContext(ctx context.Context) *websocket.Conn {
	conn := ctx.Value("wsConn")
	if conn == nil {
		return nil
	}
	return conn.(*websocket.Conn)
}

func ContextWithWSMessageID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, "wsMessageID", id)
}

func WSMessageIDFromContext(ctx context.Context) string {
	id := ctx.Value("wsMessageID")
	if id == nil {
		return ""
	}
	return id.(string)
}
