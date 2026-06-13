package mqtt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/kevinball/ares-bib-logger/backend/internal/config"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

// pahoClient is the subset of pahomqtt.Client we use, enabling mock injection in tests.
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

// pahoPublisher wraps a pahoClient to implement mqttPublisher.
type pahoPublisher struct {
	client pahoClient
}

func (p *pahoPublisher) Publish(topic string, payload []byte) error {
	tok := p.client.Publish(topic, 0, false, payload)
	tok.Wait()
	return tok.Error()
}

// Adapter is the MQTT driven adapter for Meshtastic bib input.
type Adapter struct {
	publisher mqttPublisher
	stopFn    func()
	cfg       config.MQTTConfig
	svc       portsvc.CheckpointLogService
}

func newAdapter(publisher mqttPublisher, stopFn func(), cfg config.MQTTConfig, svc portsvc.CheckpointLogService) *Adapter {
	return &Adapter{publisher: publisher, stopFn: stopFn, cfg: cfg, svc: svc}
}

// newFromClient wires up subscription on an already-constructed paho client.
// It is the testable inner constructor; New() is the public convenience wrapper.
func newFromClient(client pahoClient, cfg config.MQTTConfig, svc portsvc.CheckpointLogService) (*Adapter, error) {
	if tok := client.Connect(); tok.Wait() && tok.Error() != nil {
		return nil, fmt.Errorf("connecting to MQTT broker: %w", tok.Error())
	}

	a := newAdapter(&pahoPublisher{client: client}, func() { client.Disconnect(250) }, cfg, svc)

	topic := cfg.SubscribeTopic()
	tok := client.Subscribe(topic, 0, func(_ pahomqtt.Client, msg pahomqtt.Message) {
		a.processMessage(context.Background(), msg.Payload())
	})
	if tok.Wait() && tok.Error() != nil {
		client.Disconnect(250)
		return nil, fmt.Errorf("subscribing to %q: %w", topic, tok.Error())
	}

	slog.Info("MQTT adapter started", "topic", topic)
	return a, nil
}

// New connects to the MQTT broker and returns a running Adapter. Call Stop() to disconnect.
func New(cfg config.MQTTConfig, svc portsvc.CheckpointLogService) (*Adapter, error) {
	opts := pahomqtt.NewClientOptions().
		AddBroker(fmt.Sprintf("tcp://%s:%d", cfg.Host, cfg.Port)).
		SetClientID("ares-bib-logger").
		SetCleanSession(true)
	return newFromClient(pahomqtt.NewClient(opts), cfg, svc)
}

// Stop disconnects from the MQTT broker.
func (a *Adapter) Stop() {
	a.stopFn()
}

// serviceEnvelope mirrors the Meshtastic MQTT JSON ServiceEnvelope.
type serviceEnvelope struct {
	From    uint64 `json:"from"`
	To      uint64 `json:"to"`
	Channel int    `json:"channel"`
	ID      uint64 `json:"id"`
	RxTime  int64  `json:"rxTime"`
	Type    string `json:"type"`
	Payload struct {
		Text string `json:"text"`
	} `json:"payload"`
}

// processMessage parses a raw MQTT payload and logs any bibs found in it.
func (a *Adapter) processMessage(ctx context.Context, raw []byte) {
	var env serviceEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		slog.Warn("mqtt: ignoring unparseable message", "error", err)
		return
	}

	if env.Type != "text" {
		return
	}

	for _, bib := range parseBibs(env.Payload.Text) {
		result, err := a.svc.LogBib(ctx, portsvc.LogBibInput{
			BibNumber:  bib,
			Source:     entity.SourceMeshtastic,
			RawMessage: string(raw),
		})
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrNoSession):
				slog.Warn("mqtt: no active session, dropping bib", "bib", bib)
			case errors.Is(err, domain.ErrNotFound):
				slog.Info("mqtt: unknown bib, not in roster", "bib", bib)
			default:
				slog.Error("mqtt: error logging bib", "bib", bib, "error", err)
			}
			continue
		}

		if result.IsDuplicate {
			slog.Info("mqtt: duplicate bib, alerting mesh", "bib", bib)
			a.publishDuplicateAlert(bib)
		} else {
			slog.Info("mqtt: bib logged",
				"bib", bib,
				"runner", fmt.Sprintf("%s %s", result.Runner.FirstName, result.Runner.LastName),
				"checkpoint", result.Log.CheckpointID,
			)
		}
	}
}

// publishDuplicateAlert sends a warning back through the gateway to the mesh.
func (a *Adapter) publishDuplicateAlert(bib int) {
	nodeID := a.cfg.GatewayNodeID
	nodeUint, _ := strconv.ParseUint(nodeID, 16, 32)

	alert := map[string]any{
		"channel_id": a.cfg.ChannelName,
		"gateway_id": "!" + nodeID,
		"packet": map[string]any{
			"from": uint32(nodeUint),
			"to":   uint32(4294967295),
			"decoded": map[string]any{
				"portnum": 1,
				"payload": fmt.Sprintf("DUPLICATE BIB: %d", bib),
			},
		},
	}

	// json.Marshal cannot fail for this fixed structure of basic types.
	b, _ := json.Marshal(alert)

	if err := a.publisher.Publish(a.cfg.PublishTopic(), b); err != nil {
		slog.Error("mqtt: failed to publish duplicate alert", "bib", bib, "error", err)
	}
}

// parseBibs splits text on newlines and returns all integer values found.
// Non-numeric and empty lines are silently skipped.
func parseBibs(text string) []int {
	var bibs []int
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			slog.Debug("mqtt: skipping non-numeric bib line", "line", line)
			continue
		}
		bibs = append(bibs, n)
	}
	return bibs
}
