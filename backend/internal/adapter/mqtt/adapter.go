package mqtt

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strconv"
	"strings"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"google.golang.org/protobuf/proto"

	meshtastic "buf.build/gen/go/meshtastic/protobufs/protocolbuffers/go/meshtastic"

	"github.com/kevinball/ares-bib-logger/backend/internal/adapter/sse"
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
	// QoS 1 matches the gateway's subscription QoS for reliable delivery.
	tok := p.client.Publish(topic, 1, false, payload)
	tok.Wait()
	return tok.Error()
}

// Adapter is the MQTT driven adapter for Meshtastic bib input.
type Adapter struct {
	publisher  mqttPublisher
	stream     sse.Publisher
	stopFn     func()
	cfg        config.MQTTConfig
	svc        portsvc.CheckpointLogService
	selfNodeID uint32 // parsed from cfg.GatewayNodeID; used to drop our own echoed messages
}

func newAdapter(publisher mqttPublisher, stream sse.Publisher, stopFn func(), cfg config.MQTTConfig, svc portsvc.CheckpointLogService) *Adapter {
	nodeUint, _ := strconv.ParseUint(cfg.GatewayNodeID, 16, 32)
	return &Adapter{
		publisher:  publisher,
		stream:     stream,
		stopFn:     stopFn,
		cfg:        cfg,
		svc:        svc,
		selfNodeID: uint32(nodeUint),
	}
}

// newFromClient wires up subscription on an already-constructed paho client.
// It is the testable inner constructor; New() is the public convenience wrapper.
func newFromClient(client pahoClient, cfg config.MQTTConfig, svc portsvc.CheckpointLogService, stream sse.Publisher) (*Adapter, error) {
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

	slog.Info("MQTT adapter started", "topic", topic)
	a.publishNodeInfo()
	return a, nil
}

// New connects to the MQTT broker and returns a running Adapter. Call Stop() to disconnect.
func New(cfg config.MQTTConfig, svc portsvc.CheckpointLogService, stream sse.Publisher) (*Adapter, error) {
	opts := pahomqtt.NewClientOptions().
		AddBroker(fmt.Sprintf("tcp://%s:%d", cfg.Host, cfg.Port)).
		SetClientID("ares-bib-logger").
		SetCleanSession(true)
	return newFromClient(pahomqtt.NewClient(opts), cfg, svc, stream)
}

// Stop disconnects from the MQTT broker.
func (a *Adapter) Stop() {
	a.stopFn()
}

// alertFromNodeID and alertGatewayID identify our own downlink messages.
// Both must differ from the gateway's own node ID — the firmware filters MQTT messages
// where either the from field or the ServiceEnvelope gateway_id matches its own ID,
// silently dropping them as "downlink we originally sent."
// We also use alertGatewayID to recognise and drop our own ack echoes on the uplink subscription.
const alertFromNodeID uint32 = 1
const alertGatewayID = "00000001" // protoEncodeDownlink prepends "!"

// processMessage decodes a binary Meshtastic ServiceEnvelope and logs any bib numbers found.
func (a *Adapter) processMessage(ctx context.Context, raw []byte) {
	slog.Debug("mqtt: message received", "bytes", len(raw))

	var env meshtastic.ServiceEnvelope
	if err := proto.Unmarshal(raw, &env); err != nil {
		slog.Warn("mqtt: ignoring unparseable protobuf message", "error", err)
		return
	}

	pkt := env.GetPacket()
	// Ignore uplink echoes of our own messages.
	if pkt.GetFrom() == a.selfNodeID {
		slog.Debug("mqtt: dropping self-originated message", "from", fmt.Sprintf("0x%08x", pkt.GetFrom()))
		return
	}
	// Ignore broker echoes of our own ack/downlink messages (gateway_id is our sentinel value).
	if env.GetGatewayId() == "!"+alertGatewayID {
		slog.Debug("mqtt: dropping our own ack echo")
		return
	}
	slog.Debug("mqtt: envelope decoded",
		"channel_id", env.GetChannelId(),
		"gateway_id", env.GetGatewayId(),
		"from", fmt.Sprintf("0x%08x", pkt.GetFrom()),
	)

	decoded := pkt.GetDecoded()
	if decoded == nil {
		slog.Debug("mqtt: dropping encrypted packet — no channel key")
		return
	}

	portnum := decoded.GetPortnum()
	if portnum != meshtastic.PortNum_TEXT_MESSAGE_APP {
		slog.Debug("mqtt: dropping non-text packet", "portnum", portnum)
		return
	}

	text := string(decoded.GetPayload())
	slog.Debug("mqtt: text message received", "text", text)

	var loggedBibs, duplicateBibs []int
	for _, bib := range parseBibs(text) {
		result, err := a.svc.LogBib(ctx, portsvc.LogBibInput{
			BibNumber:  bib,
			Source:     entity.SourceMeshtastic,
			RawMessage: base64.StdEncoding.EncodeToString(raw),
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

		a.stream.Publish("bib_logged", map[string]any{
			"runner":       result.Runner,
			"log":          result.Log,
			"is_duplicate": result.IsDuplicate,
		})
		if result.IsDuplicate {
			slog.Info("mqtt: duplicate bib", "bib", bib)
			duplicateBibs = append(duplicateBibs, bib)
		} else {
			slog.Info("mqtt: bib logged",
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

// publishNodeInfo broadcasts a NODEINFO_APP packet so the mesh displays the logger with a friendly name.
func (a *Adapter) publishNodeInfo() {
	packetID := rand.Uint32()
	if packetID == 0 {
		packetID = 1
	}
	b := protoEncodeNodeInfoPkt(alertFromNodeID, a.cfg.ChannelIndex, packetID, a.cfg.ChannelName, alertGatewayID, a.cfg.NodeLongName, a.cfg.NodeShortName)
	slog.Debug("mqtt: publishing node info", "long_name", a.cfg.NodeLongName, "short_name", a.cfg.NodeShortName)
	if err := a.publisher.Publish(a.cfg.PublishTopic(), b); err != nil {
		slog.Error("mqtt: failed to publish node info", "error", err)
	}
}

// publishAck sends a single ack message to the mesh summarising all bibs from one incoming message.
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

	packetID := rand.Uint32()
	if packetID == 0 {
		packetID = 1
	}

	b := protoEncodeDownlink(
		alertFromNodeID,
		a.cfg.ChannelIndex,
		packetID,
		a.cfg.ChannelName,
		alertGatewayID,
		text,
	)

	slog.Debug("mqtt: publishing ack",
		"topic", a.cfg.PublishTopic(),
		"text", text,
		"packet_id", fmt.Sprintf("0x%08x", packetID),
		"payload_hex", hex.EncodeToString(b),
	)

	if err := a.publisher.Publish(a.cfg.PublishTopic(), b); err != nil {
		slog.Error("mqtt: failed to publish ack", "error", err)
	}
}

// protoEncodeDownlink hand-encodes a Meshtastic ServiceEnvelope for MQTT downlink.
// Fields are written in strict proto field-number order — the Go protobuf library
// places oneof fields after higher-numbered regular fields, which confuses nanopb.
// hop_start must equal hop_limit for a fresh packet.
// gateway_id MUST be non-empty: the firmware's DecodedServiceEnvelope wrapper sets the
// pointer to NULL for empty strings, and onReceiveProto rejects envelopes with a NULL
// gateway_id even though it is optional in the proto schema.
func protoEncodeDownlink(from, channelIndex, id uint32, channelID, gatewayID, text string) []byte {
	// Data: portnum(1) + payload(2)
	var data []byte
	data = pbVarintField(data, 1, 1) // portnum = TEXT_MESSAGE_APP
	data = pbBytesField(data, 2, []byte(text))

	// MeshPacket: fields in strict field-number order
	var pkt []byte
	pkt = pbFixed32Field(pkt, 1, from)
	pkt = pbFixed32Field(pkt, 2, 0xFFFFFFFF) // broadcast
	if channelIndex != 0 {
		pkt = pbVarintField(pkt, 3, uint64(channelIndex))
	}
	pkt = pbBytesField(pkt, 4, data) // decoded Data (oneof field 4)
	pkt = pbFixed32Field(pkt, 6, id)
	pkt = pbVarintField(pkt, 9, 3)  // hop_limit
	pkt = pbVarintField(pkt, 15, 3) // hop_start — must equal hop_limit for a fresh packet

	// ServiceEnvelope: packet(1), channel_id(2), gateway_id(3)
	var env []byte
	env = pbBytesField(env, 1, pkt)
	env = pbBytesField(env, 2, []byte(channelID))
	env = pbBytesField(env, 3, []byte("!"+gatewayID))
	return env
}

// protoEncodeNodeInfoPkt hand-encodes a NODEINFO_APP ServiceEnvelope for MQTT downlink.
// The User payload identifies the logger node with a human-readable name on the mesh.
func protoEncodeNodeInfoPkt(from, channelIndex, id uint32, channelID, gatewayID, longName, shortName string) []byte {
	// User: id(1), long_name(2), short_name(3)
	var user []byte
	user = pbBytesField(user, 1, []byte(fmt.Sprintf("!%08x", from)))
	user = pbBytesField(user, 2, []byte(longName))
	user = pbBytesField(user, 3, []byte(shortName))

	// Data: portnum(1) = NODEINFO_APP(4), payload(2) = User
	var data []byte
	data = pbVarintField(data, 1, 4)
	data = pbBytesField(data, 2, user)

	var pkt []byte
	pkt = pbFixed32Field(pkt, 1, from)
	pkt = pbFixed32Field(pkt, 2, 0xFFFFFFFF)
	if channelIndex != 0 {
		pkt = pbVarintField(pkt, 3, uint64(channelIndex))
	}
	pkt = pbBytesField(pkt, 4, data)
	pkt = pbFixed32Field(pkt, 6, id)
	pkt = pbVarintField(pkt, 9, 3)
	pkt = pbVarintField(pkt, 15, 3)

	var env []byte
	env = pbBytesField(env, 1, pkt)
	env = pbBytesField(env, 2, []byte(channelID))
	env = pbBytesField(env, 3, []byte("!"+gatewayID))
	return env
}

func pbVarint(v uint64) []byte {
	var b []byte
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}

func pbVarintField(dst []byte, field int, v uint64) []byte {
	dst = append(dst, pbVarint(uint64(field<<3))...) // wire type 0
	return append(dst, pbVarint(v)...)
}

func pbFixed32Field(dst []byte, field int, v uint32) []byte {
	dst = append(dst, pbVarint(uint64(field<<3|5))...) // wire type 5
	return append(dst, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

func pbBytesField(dst []byte, field int, data []byte) []byte {
	dst = append(dst, pbVarint(uint64(field<<3|2))...) // wire type 2
	dst = append(dst, pbVarint(uint64(len(data)))...)
	return append(dst, data...)
}

// parseBibs splits text on newlines, commas, and spaces, returning all integer values found.
// Non-numeric and empty tokens are silently skipped.
func parseBibs(text string) []int {
	var bibs []int
	for _, tok := range strings.FieldsFunc(text, func(r rune) bool {
		return r == '\n' || r == ',' || r == ' '
	}) {
		n, err := strconv.Atoi(tok)
		if err != nil {
			slog.Debug("mqtt: skipping non-numeric token", "token", tok)
			continue
		}
		bibs = append(bibs, n)
	}
	return bibs
}
