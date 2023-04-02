package portforward

import (
	"encoding/json"
	"fmt"
	"time"
)

// NOTICE: below types should be in sync with mono/services/ws-hub/app/internal/handlers/messageUtils.go, any changes should be backward-compatible

type MessageType string

const (
	MTPodExecInit     MessageType = "pod_exec_init"
	MTPortForwardInit MessageType = "port_forward_init"
	MTStdin           MessageType = "stdin"
	MTStdout          MessageType = "stdout"
	MTTermination     MessageType = "termination"
	MTTerminalSize    MessageType = "terminal-size"
	MTKeepAlive       MessageType = "keep-alive"
	MTAck             MessageType = "ack"
	MTPing            MessageType = "ping"
	MTError           MessageType = "error"
)

type SessionMessage struct {
	MessageId   string      `json:"messageId"`
	SessionId   string      `json:"sessionId"`
	MessageType MessageType `json:"messageType"`
	Data        interface{} `json:"data"`
	Timestamp   time.Time   `json:"timestamp"`
}

func (m *SessionMessage) UnmarshalJSON(b []byte) error {
	m.Data = &json.RawMessage{} // m.Data has to be pointer, to retain the type. interface{} is a pointer type, too

	type msgProxy SessionMessage // msgProxy will use default unmarshal, to avoid infinite recursion
	if err := json.Unmarshal(b, (*msgProxy)(m)); err != nil {
		return err
	}

	dataRaw := json.RawMessage{}
	hasData := m.Data != nil
	if hasData {
		dataRaw = *m.Data.(*json.RawMessage) // remember the raw data
	}

	// find the right type
	switch m.MessageType {
	case MTPodExecInit:
		m.Data = &WSPodExecInitData{}
	case MTPortForwardInit:
		m.Data = &WSPortForwardInitData{}
	case MTStdin:
		m.Data = &WSStdinData{}
	case MTStdout:
		m.Data = &WSStdoutData{}
	case MTTerminalSize:
		m.Data = &WSTerminalSizeData{}
	case MTTermination:
		m.Data = &WSSessionTerminationData{}
	case MTKeepAlive:
		m.Data = &WSKeepaliveData{}
	case MTAck:
		m.Data = &WSAckData{}
	case MTPing:
		m.Data = &WSPingData{}
	case MTError:
		m.Data = &WSErrorData{}
	default:
		return fmt.Errorf("unsupported message type %s", m.MessageType)
	}

	// do type-specific Data unmarshal
	if hasData {
		err := json.Unmarshal(dataRaw, &m.Data)
		return err
	}
	return nil
}

type WSPodExecInitData struct {
	Namespace     string `json:"namespace"`
	PodName       string `json:"podName"`
	ContainerName string `json:"containerName"`
	Cmd           string `json:"cmd"`
}

type WSPortForwardInitData struct {
	Namespace string `json:"namespace"`
	PodName   string `json:"podName"`
	Port      int    `json:"port"`
}

type WSStdinData struct {
	Input string `json:"input"`
}

type WSStdoutData struct {
	Out string `json:"out"`
}

type WSSessionTerminationData struct {
	ProcessExitCode int    `json:"processExitCode"`
	ExitMessage     string `json:"exitMessage"`
}

type WSKeepaliveData struct {
}

type WSPingData struct {
}

type WSAckData struct {
	AckedMessageID string `json:"ackedMessageID"`
}

type WSTerminalSizeData struct { // https://pkg.go.dev/k8s.io/client-go/tools/remotecommand#TerminalSize
	Width  uint16 `json:"width"`
	Height uint16 `json:"height"`
}

type WSErrorData struct {
	OriginalMessageID string `json:"originalMessageID"`
	ErrorMessage      string `json:"errorMessage"`
}
