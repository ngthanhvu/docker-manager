package ws

import (
	"context"
	"log"
	"net/http"
	"strings"

	"docker-ui/docker"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gorilla/websocket"
	"github.com/gorilla/mux"
)

type wsTextWriter struct {
	conn *websocket.Conn
}

func (w *wsTextWriter) Write(p []byte) (int, error) {
	if err := w.conn.WriteMessage(websocket.TextMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func LogsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
		Tail:       "300",
	}
	if tail := strings.TrimSpace(r.URL.Query().Get("tail")); tail != "" {
		options.Tail = tail
	}

	out, err := docker.Cli.ContainerLogs(context.Background(), id, options)
	if err != nil {
		log.Printf("Failed to get container logs: %v", err)
		return
	}
	defer out.Close()

	writer := &wsTextWriter{conn: conn}
	if _, err := stdcopy.StdCopy(writer, writer, out); err != nil {
		log.Printf("Log stream closed for container %s: %v", id, err)
	}
}
