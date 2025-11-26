package plugins

import (
	"context"

	"github.com/gorilla/websocket"
)

type taskIDCtxKey struct{}

type pluginDirCtxKey struct{}

func ContextWithTaskID(ctx context.Context, taskID string) context.Context {
	if taskID == "" {
		return ctx
	}

	return context.WithValue(ctx, taskIDCtxKey{}, &taskID)
}

func ContextWithPluginDir(ctx context.Context, directory string) context.Context {
	return context.WithValue(ctx, pluginDirCtxKey{}, directory)
}

func TaskIDFromContext(ctx context.Context) *string {
	taskID := ctx.Value(taskIDCtxKey{})
	if taskID == nil {
		return nil
	}
	return taskID.(*string)
}

func PluginDirFromContext(ctx context.Context) string {
	return ctx.Value(pluginDirCtxKey{}).(string)
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
