package utils

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"maps"
	"slices"

	"github.com/fatih/color"
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

	fields := make(map[string]interface{}, r.NumAttrs())

	for groupId, group := range h.attrs {
		for _, attr := range group {
			if groupId == "" {
				if _, found := slices.BinarySearch(constants.LOG_TOP_LEVEL_HIDDEN_FIELDS, attr.Key); found {
					continue
				}
				fields[attr.Key] = attr.Value.Any()
			} else {
				if m, ok := fields[groupId].(map[string]interface{}); ok {
					m[attr.Key] = attr.Value.Any()
				} else {
					fields[groupId] = map[string]interface{}{
						attr.Key: attr.Value.Any(),
					}
				}
			}
		}
	}

	r.Attrs(func(a slog.Attr) bool {
		fields[a.Key] = a.Value.Any()
		return true
	})

	var fieldsSerialized []byte
	if len(fields) != 0 {
		var err error
		fieldsSerialized, err = json.MarshalIndent(fields, "", "  ")
		if err != nil {
			slog.Error("failed to serialize fields", "error", err)
		}
	}

	timeStr := r.Time.Format("[15:05:05.000]")
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
