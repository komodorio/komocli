package portforward

import (
	"encoding/json"
	"fmt"
	"time"
)

const PortForwardCMDPrefix = "komodor-port-forward "

// NOTICE: below types should be in sync with mono/services/ws-hub/app/internal/handlers/messageUtils.go, any changes should be backward-compatible

type MessageType string

const (
	MTPodExecInit  MessageType = "pod_exec_init"
	MTStdin                    = "stdin"
	MTStdout                   = "stdout"
	MTTermination              = "termination"
	MTTerminalSize             = "terminal-size"
	MTKeepAlive                = "keep-alive"
	MTAck                      = "ack"
	MTPing                     = "ping"
	MTError                    = "error"
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
		m.Data = &PodExecInitData{}
	case MTStdin:
		m.Data = &PodExecStdinData{}
	case MTStdout:
		m.Data = &PodExecStdoutData{}
	case MTTerminalSize:
		m.Data = &PodExecTerminalSizeData{}
	case MTTermination:
		m.Data = &PodExecSessionTerminationData{}
	case MTKeepAlive:
		m.Data = &PodExecKeepaliveData{}
	case MTAck:
		m.Data = &PodExecAckData{}
	case MTPing:
		m.Data = &PodExecPingData{}
	case MTError:
		m.Data = &PodExecErrorData{}
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

type PodExecInitData struct {
	Namespace     string `json:"namespace"`
	PodName       string `json:"podName"`
	ContainerName string `json:"containerName"`
	Cmd           string `json:"cmd"`
}

type PodExecStdinData struct {
	Input string `json:"input"`
}

type PodExecStdoutData struct {
	Out string `json:"out"`
}

type PodExecSessionTerminationData struct {
	ProcessExitCode int    `json:"processExitCode"`
	ExitMessage     string `json:"exitMessage"`
}

type PodExecKeepaliveData struct {
}

type PodExecPingData struct {
}

type PodExecAckData struct {
	AckedMessageID string `json:"ackedMessageID"`
}

type PodExecTerminalSizeData struct { // https://pkg.go.dev/k8s.io/client-go/tools/remotecommand#TerminalSize
	Width  uint16 `json:"width"`
	Height uint16 `json:"height"`
}

type PodExecErrorData struct {
	OriginalMessageID string `json:"originalMessageID"`
	ErrorMessage      string `json:"errorMessage"`
}
