package utils

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"maps"
	"slices"
	"sync"

	"github.com/fatih/color"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tez-capital/tezpay/constants"
)

type PrettyHandlerOptions struct {
	slog.HandlerOptions
}

type PrettyTextLogHandler struct {
	slog.Handler
	l *log.Logger

	attrs  map[string][]slog.Attr
	groups []string
}

func isHiddenAttr(attr slog.Attr) bool {
	_, found := slices.BinarySearch(constants.LOG_TOP_LEVEL_HIDDEN_FIELDS, attr.Key)
	return found
}

func (h *PrettyTextLogHandler) Handle(ctx context.Context, r slog.Record) error {
	level := r.Level.String() + ":"
	switch r.Level {
	case slog.LevelDebug:
		level = color.MagentaString(level)
	case slog.LevelInfo:
		level = color.BlueString(level)
	case slog.LevelWarn:
		level = color.YellowString(level)
	case slog.LevelError:
		level = color.RedString(level)
	}

	fields := make(map[string]any, r.NumAttrs())

	for groupId, group := range h.attrs {
		for _, attr := range group {
			if groupId == "" {
				if isHiddenAttr(attr) {
					delete(fields, attr.Key)
					continue
				}
				fields[attr.Key] = attr.Value.Any()
			} else {
				if m, ok := fields[groupId].(map[string]any); ok {
					m[attr.Key] = attr.Value.Any()
				} else {
					fields[groupId] = map[string]any{
						attr.Key: attr.Value.Any(),
					}
				}
			}
		}
	}

	r.Attrs(func(a slog.Attr) bool {
		if !isHiddenAttr(a) {
			fields[a.Key] = a.Value.Any()
		}
		return true
	})

	var fieldsSerialized []byte
	if len(fields) != 0 {
		var err error
		fieldsSerialized, err = json.MarshalIndent(fields, "", "  ")
		if err != nil {
			slog.Error("failed to serialize fields", "error", err.Error())
		}
	}

	timeStr := r.Time.Format("[15:04:05.000]")
	msg := color.HiWhiteString(r.Message)

	h.l.Println(timeStr, level, msg, color.WhiteString(string(fieldsSerialized)))

	return nil
}

func (h *PrettyTextLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := maps.Clone(h.attrs)
	groupId := ""
	if len(h.groups) != 0 {
		groupId = h.groups[len(h.groups)-1]
	}
	newAttrs[groupId] = append(newAttrs[groupId], attrs...)

	return &PrettyTextLogHandler{
		Handler: h.Handler.WithAttrs(attrs),
		l:       h.l,
		attrs:   newAttrs,
		groups:  slices.Clone(h.groups),
	}
}

func (h *PrettyTextLogHandler) WithGroup(name string) slog.Handler {
	return &PrettyTextLogHandler{
		Handler: h.Handler.WithGroup(name),
		l:       h.l,
		attrs:   maps.Clone(h.attrs),
		groups:  append(h.groups, name),
	}
}

func NewPrettyTextLogHandler(
	out io.Writer,
	opts PrettyHandlerOptions,
) *PrettyTextLogHandler {
	h := &PrettyTextLogHandler{
		Handler: slog.NewJSONHandler(out, &opts.HandlerOptions),
		l:       log.New(out, "", 0),
		attrs:   make(map[string][]slog.Attr),
	}

	return h
}

type logServerClient struct {
	LogChannel chan string
	Ctx        context.Context
}

type LogServer struct {
	clients        map[uuid.UUID]logServerClient
	clientMtx      sync.RWMutex
	cachedLines    []string
	cachedLinesMtx sync.RWMutex
}

func (s *LogServer) AddClient(clientID uuid.UUID, ch logServerClient) {
	s.clientMtx.Lock()
	defer s.clientMtx.Unlock()
	s.clients[clientID] = ch
}

func (s *LogServer) RemoveClient(clientID uuid.UUID) {
	s.clientMtx.Lock()
	defer s.clientMtx.Unlock()
	delete(s.clients, clientID)
}

func (s *LogServer) Write(p []byte) (n int, err error) {
	s.clientMtx.RLock()
	defer s.clientMtx.RUnlock()
	s.cachedLinesMtx.Lock()
	defer s.cachedLinesMtx.Unlock()

	if len(s.cachedLines) < constants.LOG_SERVER_CACHE_CAPACITY {
		s.cachedLines = append(s.cachedLines, string(p))
	} else {
		s.cachedLines = append(s.cachedLines[1:], string(p))
	}

	for _, client := range s.clients {
		go func() {
			select {
			case client.LogChannel <- string(p):
			case <-client.Ctx.Done(): // client disconnected
			}
		}()
	}
	return len(p), nil
}

func NewLogServer(address string) *LogServer {
	logServer := &LogServer{
		clients:   make(map[uuid.UUID]logServerClient),
		clientMtx: sync.RWMutex{},
	}

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Get("/logs", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		c.Set("Transfer-Encoding", "chunked")

		client := logServerClient{
			LogChannel: make(chan string),
			Ctx:        c.Context(),
		}
		clientID, err := uuid.NewV7()
		if err != nil {
			slog.Error("failed to generate client ID", "error", err.Error())
			return err
		}
		logServer.AddClient(clientID, client)
		slog.Debug("new client connected", "clientID", clientID)

		context := c.Context()
		context.SetBodyStreamWriter(func(w *bufio.Writer) {
			logServer.cachedLinesMtx.RLock()
			for _, line := range logServer.cachedLines {
				if _, err := fmt.Fprintf(w, "data: %v\n\n", line); err != nil {
					return
				}
			}
			w.Flush()
			logServer.cachedLinesMtx.RUnlock()

			for logMessage := range client.LogChannel {
				select {
				case <-context.Done():
					logServer.RemoveClient(clientID)
					close(client.LogChannel)
					return
				default:
				}

				if _, err := fmt.Fprintf(w, "data: %v\n\n", logMessage); err != nil {
					return
				}
				w.Flush()
			}
		})

		return nil
	})

	go func() {
		slog.Info("starting log server", "address", address)
		err := app.Listen(address)
		if err != nil {
			slog.Error("log server failed", "error", err.Error())
		}
	}()

	return logServer
}

type MultiWriter struct {
	writers []io.Writer
}

func (m *MultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range m.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
	}
	return
}

func NewMultiWriter(writers ...io.Writer) *MultiWriter {
	return &MultiWriter{
		writers: writers,
	}
}

type SlogMultiHandler struct {
	handlers []slog.Handler
}

func NewSlogMultiHandler(handlers ...slog.Handler) *SlogMultiHandler {
	return &SlogMultiHandler{
		handlers: handlers,
	}
}

func (h *SlogMultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if !handler.Enabled(ctx, r.Level) {
			continue
		}
		if err := handler.Handle(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (h *SlogMultiHandler) Enabled(_ context.Context, level slog.Level) bool {
	return true
}

func (h *SlogMultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		newHandlers = append(newHandlers, handler.WithAttrs(attrs))
	}
	return &SlogMultiHandler{
		handlers: newHandlers,
	}
}

func (h *SlogMultiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		newHandlers = append(newHandlers, handler.WithGroup(name))
	}
	return &SlogMultiHandler{
		handlers: newHandlers,
	}
}
