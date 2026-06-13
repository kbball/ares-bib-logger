package mqtt

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/config"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

// --- paho mocks ---

type mockToken struct{ err error }

func (m *mockToken) Wait() bool                        { return true }
func (m *mockToken) WaitTimeout(_ time.Duration) bool  { return true }
func (m *mockToken) Done() <-chan struct{}              { ch := make(chan struct{}); close(ch); return ch }
func (m *mockToken) Error() error                      { return m.err }

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
func (m *mockPahoClient) Disconnect(_ uint)  { m.disconnected = true }
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

// envelope serialises a ServiceEnvelope for use in processMessage tests.
func envelope(typ string, text string) []byte {
	env := serviceEnvelope{Type: typ}
	env.Payload.Text = text
	b, _ := json.Marshal(env)
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

	a.processMessage(context.Background(), envelope("text", "42\n"))

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

	a.processMessage(context.Background(), envelope("text", "101\n202\n"))

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

	a.processMessage(context.Background(), envelope("text", "42\n"))

	require.Len(t, pub.published, 1)
	assert.Equal(t, testCfg().PublishTopic(), pub.published[0].topic)

	var alert map[string]any
	require.NoError(t, json.Unmarshal(pub.published[0].payload, &alert))
	assert.Equal(t, "LongFast", alert["channel_id"])
	assert.Equal(t, "!a3b4c5d6", alert["gateway_id"])
	decoded := alert["packet"].(map[string]any)["decoded"].(map[string]any)
	assert.Contains(t, decoded["payload"], "42")
}

func TestProcessMessage_NonTextIgnored(t *testing.T) {
	svc := &mockLogService{}
	a := newTestAdapter(svc, &mockPublisher{})

	a.processMessage(context.Background(), envelope("position", "101\n"))

	assert.Empty(t, svc.calls)
}

func TestProcessMessage_InvalidJSONIgnored(t *testing.T) {
	svc := &mockLogService{}
	a := newTestAdapter(svc, &mockPublisher{})

	a.processMessage(context.Background(), []byte("not json at all"))

	assert.Empty(t, svc.calls)
}

func TestProcessMessage_NoSession(t *testing.T) {
	svc := &mockLogService{err: domain.ErrNoSession}
	a := newTestAdapter(svc, &mockPublisher{})

	a.processMessage(context.Background(), envelope("text", "101\n"))

	assert.Len(t, svc.calls, 1)
}

func TestProcessMessage_UnknownBib(t *testing.T) {
	svc := &mockLogService{err: domain.ErrNotFound}
	a := newTestAdapter(svc, &mockPublisher{})

	a.processMessage(context.Background(), envelope("text", "999\n"))

	assert.Len(t, svc.calls, 1)
}

func TestProcessMessage_ServiceError(t *testing.T) {
	svc := &mockLogService{err: errors.New("db down")}
	pub := &mockPublisher{}
	a := newTestAdapter(svc, pub)

	a.processMessage(context.Background(), envelope("text", "101\n"))

	assert.Len(t, svc.calls, 1)
	assert.Empty(t, pub.published)
}

func TestProcessMessage_MultipleBibsOneBad(t *testing.T) {
	svc := &mockLogService{result: portsvc.LogBibResult{Runner: entity.Runner{BibNumber: 101}}}
	a := newTestAdapter(svc, &mockPublisher{})

	a.processMessage(context.Background(), envelope("text", "101\nabc\n202\n"))

	assert.Len(t, svc.calls, 2)
	assert.Equal(t, 101, svc.calls[0].BibNumber)
	assert.Equal(t, 202, svc.calls[1].BibNumber)
}

func TestProcessMessage_RawMessageStored(t *testing.T) {
	svc := &mockLogService{}
	a := newTestAdapter(svc, &mockPublisher{})

	raw := envelope("text", "101\n")
	a.processMessage(context.Background(), raw)

	require.Len(t, svc.calls, 1)
	assert.Equal(t, string(raw), svc.calls[0].RawMessage)
}

// --- publishDuplicateAlert ---

func TestPublishDuplicateAlert_PublishError(t *testing.T) {
	pub := &mockPublisher{err: errors.New("broker gone")}
	a := newTestAdapter(&mockLogService{}, pub)

	a.publishDuplicateAlert(42)

	assert.Len(t, pub.published, 1)
}

// --- parseBibs ---

func TestParseBibs_Mixed(t *testing.T) {
	bibs := parseBibs("101\n\nabc\n202\n  303  \n")
	assert.Equal(t, []int{101, 202, 303}, bibs)
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
