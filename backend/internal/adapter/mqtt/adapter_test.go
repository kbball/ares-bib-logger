package mqtt

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	meshtastic "buf.build/gen/go/meshtastic/protobufs/protocolbuffers/go/meshtastic"

	"github.com/kevinball/ares-bib-logger/backend/internal/config"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

// --- paho mocks ---

type mockToken struct{ err error }

func (m *mockToken) Wait() bool                       { return true }
func (m *mockToken) WaitTimeout(_ time.Duration) bool { return true }
func (m *mockToken) Done() <-chan struct{}            { ch := make(chan struct{}); close(ch); return ch }
func (m *mockToken) Error() error                     { return m.err }

type mockPahoClient struct {
	connectErr   error
	subscribeErr error
	published    [][]byte
	disconnected bool
}

func (m *mockPahoClient) Connect() pahomqtt.Token { return &mockToken{m.connectErr} }
func (m *mockPahoClient) Subscribe(_ string, _ byte, _ pahomqtt.MessageHandler) pahomqtt.Token {
	return &mockToken{m.subscribeErr}
}
func (m *mockPahoClient) Disconnect(_ uint) { m.disconnected = true }
func (m *mockPahoClient) Publish(_ string, _ byte, _ bool, payload any) pahomqtt.Token {
	if b, ok := payload.([]byte); ok {
		m.published = append(m.published, b)
	}
	return &mockToken{}
}

// --- service / publisher mocks ---

type mockLogService struct {
	result portsvc.LogBibResult
	err    error
	calls  []portsvc.LogBibInput
}

func (m *mockLogService) LogBib(_ context.Context, input portsvc.LogBibInput) (portsvc.LogBibResult, error) {
	m.calls = append(m.calls, input)
	return m.result, m.err
}

func (m *mockLogService) LogStatus(_ context.Context, _ int, _ entity.RunnerStatus) error {
	return nil
}

func (m *mockLogService) ListByRace(_ context.Context, _ int) ([]entity.CheckpointLog, error) {
	return nil, nil
}

type mockPublisher struct {
	published []struct {
		topic   string
		payload []byte
	}
	err error
}

func (m *mockPublisher) Publish(topic string, payload []byte) error {
	m.published = append(m.published, struct {
		topic   string
		payload []byte
	}{topic, payload})
	return m.err
}

type mockSSEPublisher struct {
	events []struct {
		eventType string
		payload   any
	}
}

func (m *mockSSEPublisher) Publish(eventType string, payload any) {
	m.events = append(m.events, struct {
		eventType string
		payload   any
	}{eventType, payload})
}

// --- helpers ---

func testCfg() config.MQTTConfig {
	return config.MQTTConfig{
		Region:        "US",
		ChannelNum:    2,
		ChannelName:   "LongFast",
		GatewayNodeID: "a3b4c5d6",
	}
}

// newTestAdapter builds an adapter with mock publisher (bypasses paho entirely).
func newTestAdapter(svc *mockLogService, pub *mockPublisher) *Adapter {
	return newAdapter(pub, &mockSSEPublisher{}, func() {}, testCfg(), svc)
}

// textEnvelopeFrom serialises a ServiceEnvelope with a specific From node ID.
func textEnvelopeFrom(text string, from uint32) []byte {
	env := &meshtastic.ServiceEnvelope{
		ChannelId: "LongFast",
		GatewayId: "!a3b4c5d6",
		Packet: &meshtastic.MeshPacket{
			From: from,
			PayloadVariant: &meshtastic.MeshPacket_Decoded{
				Decoded: &meshtastic.Data{
					Portnum: meshtastic.PortNum_TEXT_MESSAGE_APP,
					Payload: []byte(text),
				},
			},
		},
	}
	b, _ := proto.Marshal(env)
	return b
}

// textEnvelope serialises a ServiceEnvelope containing a TEXT_MESSAGE_APP payload.
// From is set to a non-zero node ID to simulate a real mesh sender (not our own echo).
func textEnvelope(text string) []byte {
	env := &meshtastic.ServiceEnvelope{
		ChannelId: "LongFast",
		GatewayId: "!a3b4c5d6",
		Packet: &meshtastic.MeshPacket{
			From: 0x00000001,
			PayloadVariant: &meshtastic.MeshPacket_Decoded{
				Decoded: &meshtastic.Data{
					Portnum: meshtastic.PortNum_TEXT_MESSAGE_APP,
					Payload: []byte(text),
				},
			},
		},
	}
	b, _ := proto.Marshal(env)
	return b
}

// positionEnvelope serialises a ServiceEnvelope with a non-text portnum.
func positionEnvelope() []byte {
	env := &meshtastic.ServiceEnvelope{
		Packet: &meshtastic.MeshPacket{
			From: 0x00000001,
			PayloadVariant: &meshtastic.MeshPacket_Decoded{
				Decoded: &meshtastic.Data{
					Portnum: meshtastic.PortNum_POSITION_APP,
					Payload: []byte("ignored"),
				},
			},
		},
	}
	b, _ := proto.Marshal(env)
	return b
}

// --- newFromClient tests ---

func TestNewFromClient_Success(t *testing.T) {
	paho := &mockPahoClient{}
	a, err := newFromClient(paho, testCfg(), &mockLogService{}, &mockSSEPublisher{})
	require.NoError(t, err)
	assert.NotNil(t, a)
}

func TestNewFromClient_ConnectError(t *testing.T) {
	paho := &mockPahoClient{connectErr: errors.New("connection refused")}
	_, err := newFromClient(paho, testCfg(), &mockLogService{}, &mockSSEPublisher{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "connecting")
}

func TestNewFromClient_SubscribeError(t *testing.T) {
	paho := &mockPahoClient{subscribeErr: errors.New("bad topic")}
	_, err := newFromClient(paho, testCfg(), &mockLogService{}, &mockSSEPublisher{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "subscribing")
	assert.True(t, paho.disconnected, "should disconnect on subscribe failure")
}

// TestPahoPublisher_CoversPublish exercises pahoPublisher.Publish() via a duplicate alert
// triggered through newFromClient so the adapter's publisher is the real pahoPublisher.
func TestPahoPublisher_CoversPublish(t *testing.T) {
	svc := &mockLogService{result: portsvc.LogBibResult{
		Runner:      entity.Runner{ID: 1, BibNumber: 42},
		IsDuplicate: true,
	}}
	paho := &mockPahoClient{}
	a, err := newFromClient(paho, testCfg(), svc, &mockSSEPublisher{})
	require.NoError(t, err)

	a.processMessage(context.Background(), textEnvelope("42\n"))

	assert.NotEmpty(t, paho.published, "pahoPublisher.Publish should have been called")
}

// --- processMessage tests ---

func TestProcessMessage_LogsBibs(t *testing.T) {
	svc := &mockLogService{
		result: portsvc.LogBibResult{
			Runner: entity.Runner{ID: 1, BibNumber: 101, FirstName: "Alice", LastName: "Smith"},
			Log:    entity.CheckpointLog{ID: 1, CheckpointID: 5},
		},
	}
	pub := &mockPublisher{}
	a := newTestAdapter(svc, pub)

	a.processMessage(context.Background(), textEnvelope("101\n202\n"))

	require.Len(t, svc.calls, 2)
	assert.Equal(t, 101, svc.calls[0].BibNumber)
	assert.Equal(t, 202, svc.calls[1].BibNumber)
	assert.Equal(t, entity.SourceMeshtastic, svc.calls[0].Source)
	assert.Empty(t, pub.published)
}

func TestProcessMessage_DuplicatePublishesAlert(t *testing.T) {
	svc := &mockLogService{
		result: portsvc.LogBibResult{
			Runner:      entity.Runner{ID: 1, BibNumber: 42},
			IsDuplicate: true,
		},
	}
	pub := &mockPublisher{}
	a := newTestAdapter(svc, pub)

	a.processMessage(context.Background(), textEnvelope("42\n"))

	require.Len(t, pub.published, 1)
	assert.Equal(t, testCfg().PublishTopic(), pub.published[0].topic)

	var alert meshtastic.ServiceEnvelope
	require.NoError(t, proto.Unmarshal(pub.published[0].payload, &alert))
	assert.Equal(t, "LongFast", alert.GetChannelId())
	// gateway_id must be non-empty (firmware rejects NULL) but must NOT match the gateway's
	// own node ID or the firmware silently drops it as "downlink we originally sent".
	assert.Equal(t, "!00000001", alert.GetGatewayId())
	pkt := alert.GetPacket()
	assert.Equal(t, uint32(1), pkt.GetFrom()) // must not be selfNodeID — see publishDuplicateAlert
	assert.Equal(t, uint32(0xFFFFFFFF), pkt.GetTo())
	assert.NotZero(t, pkt.GetId())
	assert.Equal(t, uint32(3), pkt.GetHopLimit())
	assert.Equal(t, uint32(3), pkt.GetHopStart()) // must equal hop_limit for a fresh packet
	assert.Contains(t, string(pkt.GetDecoded().GetPayload()), "42")
}

func TestProcessMessage_NonTextIgnored(t *testing.T) {
	svc := &mockLogService{}
	a := newTestAdapter(svc, &mockPublisher{})

	a.processMessage(context.Background(), positionEnvelope())

	assert.Empty(t, svc.calls)
}

func TestProcessMessage_InvalidProtoIgnored(t *testing.T) {
	svc := &mockLogService{}
	a := newTestAdapter(svc, &mockPublisher{})

	a.processMessage(context.Background(), []byte("not valid protobuf \xff\xfe"))

	assert.Empty(t, svc.calls)
}

func TestProcessMessage_NoSession(t *testing.T) {
	svc := &mockLogService{err: domain.ErrNoSession}
	a := newTestAdapter(svc, &mockPublisher{})

	a.processMessage(context.Background(), textEnvelope("101\n"))

	assert.Len(t, svc.calls, 1)
}

func TestProcessMessage_UnknownBib(t *testing.T) {
	svc := &mockLogService{err: domain.ErrNotFound}
	a := newTestAdapter(svc, &mockPublisher{})

	a.processMessage(context.Background(), textEnvelope("999\n"))

	assert.Len(t, svc.calls, 1)
}

func TestProcessMessage_ServiceError(t *testing.T) {
	svc := &mockLogService{err: errors.New("db down")}
	pub := &mockPublisher{}
	a := newTestAdapter(svc, pub)

	a.processMessage(context.Background(), textEnvelope("101\n"))

	assert.Len(t, svc.calls, 1)
	assert.Empty(t, pub.published)
}

func TestProcessMessage_MultipleBibsOneBad(t *testing.T) {
	svc := &mockLogService{result: portsvc.LogBibResult{Runner: entity.Runner{BibNumber: 101}}}
	a := newTestAdapter(svc, &mockPublisher{})

	a.processMessage(context.Background(), textEnvelope("101\nabc\n202\n"))

	assert.Len(t, svc.calls, 2)
	assert.Equal(t, 101, svc.calls[0].BibNumber)
	assert.Equal(t, 202, svc.calls[1].BibNumber)
}

func TestProcessMessage_RawMessageStored(t *testing.T) {
	svc := &mockLogService{}
	a := newTestAdapter(svc, &mockPublisher{})

	raw := textEnvelope("101\n")
	a.processMessage(context.Background(), raw)

	require.Len(t, svc.calls, 1)
	assert.Equal(t, base64.StdEncoding.EncodeToString(raw), svc.calls[0].RawMessage)
}

func TestProcessMessage_SelfOriginatedIgnored(t *testing.T) {
	svc := &mockLogService{}
	a := newTestAdapter(svc, &mockPublisher{})

	// Simulate our own duplicate alert echoed back by the broker.
	// Our outgoing alerts use From=selfNodeID so the gateway can identify the sender.
	env := &meshtastic.ServiceEnvelope{
		Packet: &meshtastic.MeshPacket{
			From: 0xa3b4c5d6, // matches testCfg().GatewayNodeID parsed as uint32
			PayloadVariant: &meshtastic.MeshPacket_Decoded{
				Decoded: &meshtastic.Data{
					Portnum: meshtastic.PortNum_TEXT_MESSAGE_APP,
					Payload: []byte("19"),
				},
			},
		},
	}
	b, _ := proto.Marshal(env)
	a.processMessage(context.Background(), b)

	assert.Empty(t, svc.calls)
}

func TestProcessMessage_DuplicateAlertEchoIgnored(t *testing.T) {
	svc := &mockLogService{}
	a := newTestAdapter(svc, &mockPublisher{})

	// Simulate our own "DUPLICATE BIB: N" text echoed back from the radio or broker.
	a.processMessage(context.Background(), textEnvelopeFrom("DUPLICATE BIB: 42", 1))

	assert.Empty(t, svc.calls)
}

func TestProcessMessage_EncryptedPacketIgnored(t *testing.T) {
	env := &meshtastic.ServiceEnvelope{
		Packet: &meshtastic.MeshPacket{
			PayloadVariant: &meshtastic.MeshPacket_Encrypted{
				Encrypted: []byte("cipher"),
			},
		},
	}
	b, _ := proto.Marshal(env)

	svc := &mockLogService{}
	a := newTestAdapter(svc, &mockPublisher{})
	a.processMessage(context.Background(), b)

	assert.Empty(t, svc.calls)
}

// --- publishDuplicateAlert ---

func TestPublishDuplicateAlert_PublishError(t *testing.T) {
	pub := &mockPublisher{err: errors.New("broker gone")}
	a := newTestAdapter(&mockLogService{}, pub)

	a.publishDuplicateAlert(42)

	assert.Len(t, pub.published, 1)
}

// --- parseBibs ---

func TestParseBibs_Newlines(t *testing.T) {
	bibs := parseBibs("101\n\nabc\n202\n  303  \n")
	assert.Equal(t, []int{101, 202, 303}, bibs)
}

func TestParseBibs_Commas(t *testing.T) {
	assert.Equal(t, []int{101, 202, 303}, parseBibs("101,202,303"))
}

func TestParseBibs_Spaces(t *testing.T) {
	assert.Equal(t, []int{101, 202, 303}, parseBibs("101 202 303"))
}

func TestParseBibs_Mixed(t *testing.T) {
	assert.Equal(t, []int{101, 202, 303}, parseBibs("101, 202,303"))
}

func TestParseBibs_Empty(t *testing.T) {
	assert.Empty(t, parseBibs(""))
	assert.Empty(t, parseBibs("\n\n\n"))
}

func TestParseBibs_SingleBib(t *testing.T) {
	assert.Equal(t, []int{42}, parseBibs("42"))
}

// --- Stop ---

func TestStop_CallsStopFn(t *testing.T) {
	called := false
	a := newAdapter(&mockPublisher{}, &mockSSEPublisher{}, func() { called = true }, testCfg(), &mockLogService{})
	a.Stop()
	assert.True(t, called)
}
