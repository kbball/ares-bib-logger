package meshcore

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/kevinball/ares-bib-logger/backend/internal/adapter/meshutil"
	"github.com/kevinball/ares-bib-logger/backend/internal/adapter/sse"
	"github.com/kevinball/ares-bib-logger/backend/internal/config"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

// incomingMsg is the JSON payload published by meshcoreHQ/meshcore-mqtt on receive topics.
// Topic pattern: meshcore/{channel}/{sender}
type incomingMsg struct {
	From    string `json:"from"`
	Text    string `json:"text"`
	Channel string `json:"channel"`
}

// outgoingMsg is the JSON payload expected by meshcoreHQ/meshcore-mqtt on the TX topic.
// Topic: meshcore/{channel}/tx
type outgoingMsg struct {
	Text string `json:"text"`
}

// pahoClient is the subset of pahomqtt.Client used, enabling mock injection in tests.
type pahoClient interface {
	Connect() pahomqtt.Token
	Subscribe(topic string, qos byte, callback pahomqtt.MessageHandler) pahomqtt.Token
	Disconnect(quiesce uint)
	Publish(topic string, qos byte, retained bool, payload any) pahomqtt.Token
}

// mqttPublisher abstracts MQTT publish for testability.
type mqttPublisher interface {
	Publish(topic string, payload []byte) error
}

type pahoPublisher struct {
	client pahoClient
}

func (p *pahoPublisher) Publish(topic string, payload []byte) error {
	tok := p.client.Publish(topic, 1, false, payload)
	tok.Wait()
	return tok.Error()
}

// Adapter is the MeshCore MQTT driven adapter for bib input via meshcoreHQ/meshcore-mqtt.
type Adapter struct {
	publisher mqttPublisher
	stream    sse.Publisher
	stopFn    func()
	cfg       config.MeshcoreConfig
	svc       portsvc.CheckpointLogService
}

func newAdapter(publisher mqttPublisher, stream sse.Publisher, stopFn func(), cfg config.MeshcoreConfig, svc portsvc.CheckpointLogService) *Adapter {
	return &Adapter{
		publisher: publisher,
		stream:    stream,
		stopFn:    stopFn,
		cfg:       cfg,
		svc:       svc,
	}
}

func newFromClient(client pahoClient, cfg config.MeshcoreConfig, svc portsvc.CheckpointLogService, stream sse.Publisher) (*Adapter, error) {
	if tok := client.Connect(); tok.Wait() && tok.Error() != nil {
		return nil, fmt.Errorf("connecting to MQTT broker: %w", tok.Error())
	}

	a := newAdapter(&pahoPublisher{client: client}, stream, func() { client.Disconnect(250) }, cfg, svc)

	topic := cfg.SubscribeTopic()
	tok := client.Subscribe(topic, 0, func(_ pahomqtt.Client, msg pahomqtt.Message) {
		a.processMessage(context.Background(), msg.Payload())
	})
	if tok.Wait() && tok.Error() != nil {
		client.Disconnect(250)
		return nil, fmt.Errorf("subscribing to %q: %w", topic, tok.Error())
	}

	slog.Info("MeshCore adapter started", "topic", topic)
	return a, nil
}

// New connects to the MQTT broker and returns a running Adapter. Call Stop() to disconnect.
func New(cfg config.MeshcoreConfig, svc portsvc.CheckpointLogService, stream sse.Publisher) (*Adapter, error) {
	opts := pahomqtt.NewClientOptions().
		AddBroker(fmt.Sprintf("tcp://%s:%d", cfg.MQTTHost, cfg.MQTTPort)).
		SetClientID("ares-bib-logger-meshcore").
		SetCleanSession(true)
	return newFromClient(pahomqtt.NewClient(opts), cfg, svc, stream)
}

// Stop disconnects from the MQTT broker.
func (a *Adapter) Stop() {
	a.stopFn()
}

// processMessage decodes a JSON MeshCore message and logs any bib numbers found in the text.
func (a *Adapter) processMessage(ctx context.Context, raw []byte) {
	slog.Debug("meshcore: message received", "bytes", len(raw))

	var msg incomingMsg
	if err := json.Unmarshal(raw, &msg); err != nil {
		slog.Warn("meshcore: ignoring unparseable JSON message", "error", err)
		return
	}

	slog.Debug("meshcore: message decoded", "from", msg.From, "channel", msg.Channel, "text", msg.Text)

	var loggedBibs, duplicateBibs []int
	for _, bib := range meshutil.ParseBibs(msg.Text) {
		result, err := a.svc.LogBib(ctx, portsvc.LogBibInput{
			BibNumber:  bib,
			Source:     entity.SourceMeshcore,
			RawMessage: base64.StdEncoding.EncodeToString(raw),
		})
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrNoSession):
				slog.Warn("meshcore: no active session, dropping bib", "bib", bib)
			case errors.Is(err, domain.ErrNotFound):
				slog.Info("meshcore: unknown bib, not in roster", "bib", bib)
			default:
				slog.Error("meshcore: error logging bib", "bib", bib, "error", err)
			}
			continue
		}

		a.stream.Publish("bib_logged", map[string]any{
			"runner":       result.Runner,
			"log":          result.Log,
			"is_duplicate": result.IsDuplicate,
		})
		if result.IsDuplicate {
			slog.Info("meshcore: duplicate bib", "bib", bib)
			duplicateBibs = append(duplicateBibs, bib)
		} else {
			slog.Info("meshcore: bib logged",
				"bib", bib,
				"runner", fmt.Sprintf("%s %s", result.Runner.FirstName, result.Runner.LastName),
				"checkpoint", result.Log.CheckpointID,
			)
			loggedBibs = append(loggedBibs, bib)
		}
	}

	if len(loggedBibs) > 0 || len(duplicateBibs) > 0 {
		a.publishAck(loggedBibs, duplicateBibs)
	}
}

// publishAck sends an ACK message to the mesh summarising all bibs from one incoming message.
// New bibs appear as "LOGGED: N", duplicates as "DUPLICATE BIB: N", one per line.
func (a *Adapter) publishAck(loggedBibs, duplicateBibs []int) {
	var lines []string
	for _, b := range loggedBibs {
		lines = append(lines, fmt.Sprintf("LOGGED: %d", b))
	}
	for _, b := range duplicateBibs {
		lines = append(lines, fmt.Sprintf("DUPLICATE BIB: %d", b))
	}
	text := strings.Join(lines, "\n")

	payload, err := json.Marshal(outgoingMsg{Text: text})
	if err != nil {
		slog.Error("meshcore: failed to marshal ack", "error", err)
		return
	}

	topic := a.cfg.PublishTopic()
	slog.Debug("meshcore: publishing ack", "topic", topic, "text", text)

	if err := a.publisher.Publish(topic, payload); err != nil {
		slog.Error("meshcore: failed to publish ack", "error", err)
	}
}
