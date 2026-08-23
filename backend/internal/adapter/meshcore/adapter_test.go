package meshcore

import (
	"context"
	"encoding/base64"
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
	result     portsvc.LogBibResult
	err        error
	calls      []portsvc.LogBibInput
	queryText  string
	queryErr   error
	queryCalls []int
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

func (m *mockLogService) QueryRunner(_ context.Context, bibNumber int) (string, error) {
	m.queryCalls = append(m.queryCalls, bibNumber)
	return m.queryText, m.queryErr
}

func (m *mockLogService) CorrectLog(_ context.Context, _, _, _ int, _, _ string) (entity.CheckpointLog, error) {
	return entity.CheckpointLog{}, nil
}

func (m *mockLogService) DeleteLog(_ context.Context, _, _, _ int) error {
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

func testCfg() config.MeshcoreConfig {
	return config.MeshcoreConfig{
		ChannelIndex: 2,
		MQTTHost:     "localhost",
		MQTTPort:     1883,
	}
}

func newTestAdapter(svc *mockLogService, pub *mockPublisher) *Adapter {
	return newAdapter(pub, &mockSSEPublisher{}, func() {}, testCfg(), svc)
}

// jsonMsg encodes a MeshCore bridge message payload with the given text and channel index.
func jsonMsg(text string, channelIdx int) []byte {
	b, _ := json.Marshal(incomingMsg{
		Type: "EventType.CHANNEL_MSG_RECV",
		Payload: incomingPayload{
			Text:       text,
			ChannelIdx: channelIdx,
		},
	})
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

func TestPahoPublisher_CoversPublish(t *testing.T) {
	svc := &mockLogService{result: portsvc.LogBibResult{
		Runner:      entity.Runner{ID: 1, BibNumber: 42},
		IsDuplicate: true,
	}}
	paho := &mockPahoClient{}
	a, err := newFromClient(paho, testCfg(), svc, &mockSSEPublisher{})
	require.NoError(t, err)

	a.processMessage(context.Background(), jsonMsg("NODE1: 42", 2))

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

	a.processMessage(context.Background(), jsonMsg("W4KIB NODE1: 101,202", 2))

	require.Len(t, svc.calls, 2)
	assert.Equal(t, 101, svc.calls[0].BibNumber)
	assert.Equal(t, 202, svc.calls[1].BibNumber)
	assert.Equal(t, entity.SourceMeshcore, svc.calls[0].Source)
	require.Len(t, pub.published, 1)

	var ack outgoingMsg
	require.NoError(t, json.Unmarshal(pub.published[0].payload, &ack))
	assert.Equal(t, 2, ack.Channel)
	assert.Equal(t, "LOGGED: 101\nLOGGED: 202", ack.Message)
}

func TestProcessMessage_Query_PublishesReplyAndDoesNotLogBib(t *testing.T) {
	svc := &mockLogService{queryText: "101 Alice Smith: ACTIVE last AS2 11:00 pace 6:00 /mi"}
	pub := &mockPublisher{}
	a := newTestAdapter(svc, pub)

	a.processMessage(context.Background(), jsonMsg("NODE1: query 101", 2))

	require.Equal(t, []int{101}, svc.queryCalls)
	assert.Empty(t, svc.calls, "query must not fall through to bib logging")

	require.Len(t, pub.published, 1)
	var ack outgoingMsg
	require.NoError(t, json.Unmarshal(pub.published[0].payload, &ack))
	assert.Equal(t, 2, ack.Channel)
	assert.Equal(t, "101 Alice Smith: ACTIVE last AS2 11:00 pace 6:00 /mi", ack.Message)
}

func TestProcessMessage_Query_CaseInsensitive(t *testing.T) {
	svc := &mockLogService{queryText: "101 not found"}
	pub := &mockPublisher{}
	a := newTestAdapter(svc, pub)

	a.processMessage(context.Background(), jsonMsg("NODE1: QUERY 101", 2))

	require.Equal(t, []int{101}, svc.queryCalls)
	assert.Empty(t, svc.calls)
}

func TestProcessMessage_Query_ServiceErrorDoesNotPublish(t *testing.T) {
	svc := &mockLogService{queryErr: errors.New("db down")}
	pub := &mockPublisher{}
	a := newTestAdapter(svc, pub)

	a.processMessage(context.Background(), jsonMsg("NODE1: query 101", 2))

	require.Equal(t, []int{101}, svc.queryCalls)
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

	a.processMessage(context.Background(), jsonMsg("NODE1: 42", 2))

	require.Len(t, pub.published, 1)
	assert.Equal(t, testCfg().PublishTopic(), pub.published[0].topic)

	var ack outgoingMsg
	require.NoError(t, json.Unmarshal(pub.published[0].payload, &ack))
	assert.Equal(t, 2, ack.Channel)
	assert.Equal(t, "DUPLICATE BIB: 42", ack.Message)
}

func TestProcessMessage_InvalidJSONIgnored(t *testing.T) {
	svc := &mockLogService{}
	a := newTestAdapter(svc, &mockPublisher{})

	a.processMessage(context.Background(), []byte("not valid json"))

	assert.Empty(t, svc.calls)
}

func TestProcessMessage_NoSession(t *testing.T) {
	svc := &mockLogService{err: domain.ErrNoSession}
	a := newTestAdapter(svc, &mockPublisher{})

	a.processMessage(context.Background(), jsonMsg("NODE1: 101", 2))

	assert.Len(t, svc.calls, 1)
}

func TestProcessMessage_UnknownBib(t *testing.T) {
	svc := &mockLogService{err: domain.ErrNotFound}
	a := newTestAdapter(svc, &mockPublisher{})

	a.processMessage(context.Background(), jsonMsg("NODE1: 999", 2))

	assert.Len(t, svc.calls, 1)
}

func TestProcessMessage_ServiceError(t *testing.T) {
	svc := &mockLogService{err: errors.New("db down")}
	pub := &mockPublisher{}
	a := newTestAdapter(svc, pub)

	a.processMessage(context.Background(), jsonMsg("NODE1: 101", 2))

	assert.Len(t, svc.calls, 1)
	assert.Empty(t, pub.published)
}

func TestProcessMessage_MultipleBibsOneBad(t *testing.T) {
	svc := &mockLogService{result: portsvc.LogBibResult{Runner: entity.Runner{BibNumber: 101}}}
	a := newTestAdapter(svc, &mockPublisher{})

	a.processMessage(context.Background(), jsonMsg("NODE1: 101 abc 202", 2))

	assert.Len(t, svc.calls, 2)
	assert.Equal(t, 101, svc.calls[0].BibNumber)
	assert.Equal(t, 202, svc.calls[1].BibNumber)
}

func TestProcessMessage_CallsignNotParsedAsBib(t *testing.T) {
	svc := &mockLogService{result: portsvc.LogBibResult{Runner: entity.Runner{BibNumber: 17}}}
	a := newTestAdapter(svc, &mockPublisher{})

	// "W4KIB" contains "4" — must not be parsed as bib 4
	a.processMessage(context.Background(), jsonMsg("W4KIB T1000E: 17,18,23", 2))

	require.Len(t, svc.calls, 3)
	assert.Equal(t, 17, svc.calls[0].BibNumber)
	assert.Equal(t, 18, svc.calls[1].BibNumber)
	assert.Equal(t, 23, svc.calls[2].BibNumber)
}

func TestProcessMessage_RawMessageStored(t *testing.T) {
	svc := &mockLogService{}
	a := newTestAdapter(svc, &mockPublisher{})

	raw := jsonMsg("NODE1: 101", 2)
	a.processMessage(context.Background(), raw)

	require.Len(t, svc.calls, 1)
	assert.Equal(t, base64.StdEncoding.EncodeToString(raw), svc.calls[0].RawMessage)
}

func TestProcessMessage_NoAckWhenNoBibs(t *testing.T) {
	svc := &mockLogService{}
	pub := &mockPublisher{}
	a := newTestAdapter(svc, pub)

	a.processMessage(context.Background(), jsonMsg("NODE1: hello no bibs here", 2))

	assert.Empty(t, pub.published)
}

func TestProcessMessage_SSEEventPublished(t *testing.T) {
	stream := &mockSSEPublisher{}
	svc := &mockLogService{result: portsvc.LogBibResult{
		Runner: entity.Runner{ID: 1, BibNumber: 42},
		Log:    entity.CheckpointLog{ID: 1},
	}}
	a := newAdapter(&mockPublisher{}, stream, func() {}, testCfg(), svc)

	a.processMessage(context.Background(), jsonMsg("NODE1: 42", 2))

	require.Len(t, stream.events, 1)
	assert.Equal(t, "bib_logged", stream.events[0].eventType)
}

// --- publishAck ---

func TestPublishAck_LoggedOnly(t *testing.T) {
	pub := &mockPublisher{}
	a := newTestAdapter(&mockLogService{}, pub)

	a.publishAck(2, []int{101, 202}, nil)

	require.Len(t, pub.published, 1)
	var ack outgoingMsg
	require.NoError(t, json.Unmarshal(pub.published[0].payload, &ack))
	assert.Equal(t, 2, ack.Channel)
	assert.Equal(t, "LOGGED: 101\nLOGGED: 202", ack.Message)
}

func TestPublishAck_DuplicateOnly(t *testing.T) {
	pub := &mockPublisher{}
	a := newTestAdapter(&mockLogService{}, pub)

	a.publishAck(2, nil, []int{42})

	require.Len(t, pub.published, 1)
	var ack outgoingMsg
	require.NoError(t, json.Unmarshal(pub.published[0].payload, &ack))
	assert.Equal(t, "DUPLICATE BIB: 42", ack.Message)
}

func TestPublishAck_Mixed(t *testing.T) {
	pub := &mockPublisher{}
	a := newTestAdapter(&mockLogService{}, pub)

	a.publishAck(2, []int{101}, []int{42})

	require.Len(t, pub.published, 1)
	var ack outgoingMsg
	require.NoError(t, json.Unmarshal(pub.published[0].payload, &ack))
	assert.Equal(t, "LOGGED: 101\nDUPLICATE BIB: 42", ack.Message)
}

func TestPublishAck_PublishError(t *testing.T) {
	pub := &mockPublisher{err: errors.New("broker gone")}
	a := newTestAdapter(&mockLogService{}, pub)

	a.publishAck(2, []int{42}, nil)

	assert.Len(t, pub.published, 1)
}

// --- Stop ---

func TestStop_CallsStopFn(t *testing.T) {
	called := false
	a := newAdapter(&mockPublisher{}, &mockSSEPublisher{}, func() { called = true }, testCfg(), &mockLogService{})
	a.Stop()
	assert.True(t, called)
}
