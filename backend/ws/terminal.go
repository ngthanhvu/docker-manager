package ws

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"docker-ui/auth"
	"docker-ui/docker"
	"github.com/docker/docker/api/types"
	"github.com/gorilla/mux"
)

func TerminalHandler(w http.ResponseWriter, r *http.Request) {
	if RequestAuthorizer != nil {
		if err := RequestAuthorizer(r); err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, auth.ErrUnauthorized) {
				status = http.StatusUnauthorized
			}
			http.Error(w, err.Error(), status)
			return
		}
	}

	vars := mux.Vars(r)
	id := vars["id"]

	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	requestedShell := strings.TrimSpace(r.URL.Query().Get("shell"))
	shells := []string{}
	if requestedShell == "/bin/bash" || requestedShell == "/bin/sh" {
		shells = append(shells, requestedShell)
	}
	// Always keep a safe fallback so the terminal can still open.
	shells = append(shells, "/bin/sh", "/bin/bash")

	var execID types.IDResponse
	var lastErr error
	for _, shell := range shells {
		execConfig := types.ExecConfig{
			AttachStdout: true,
			AttachStderr: true,
			AttachStdin:  true,
			Tty:          true,
			Cmd:          []string{shell},
			Env:          []string{"TERM=xterm-256color"},
		}
		execID, err = docker.Cli.ContainerExecCreate(context.Background(), id, execConfig)
		if err == nil {
			lastErr = nil
			break
		}
		lastErr = err
	}
	if lastErr != nil {
		log.Printf("Failed to create exec: %v", lastErr)
		return
	}

	attachConfig := types.ExecStartCheck{
		Tty: true,
	}

	resp, err := docker.Cli.ContainerExecAttach(context.Background(), execID.ID, attachConfig)
	if err != nil {
		log.Printf("Failed to attach exec: %v", err)
		return
	}
	defer resp.Close()

	// Handle input from WebSocket to container
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if _, err := resp.Conn.Write(msg); err != nil {
				return
			}
		}
	}()

	// Handle output from container to WebSocket
	buf := make([]byte, 1024)
	for {
		n, err := resp.Reader.Read(buf)
		if n > 0 {
			if err := conn.WriteMessage(1, buf[:n]); err != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}
