package messaging

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"
)

const ProtocolVersion = "5.0"

func WireMsgToComposedMsg(msgparts [][]byte, signkey []byte) (ComposedMsg, [][]byte, error) {
	i := 0
	for string(msgparts[i]) != "<IDS|MSG>" {
		i++
	}
	identities := msgparts[:i]

	var msg ComposedMsg
	if len(signkey) != 0 {
		mac := hmac.New(sha256.New, signkey)
		for _, msgpart := range msgparts[i+2 : i+6] {
			mac.Write(msgpart)
		}
		signature := make([]byte, hex.DecodedLen(len(msgparts[i+1])))
		hex.Decode(signature, msgparts[i+1])
		if !hmac.Equal(mac.Sum(nil), signature) {
			return msg, nil, &InvalidSignatureError{}
		}
	}

	json.Unmarshal(msgparts[i+2], &msg.Header)
	json.Unmarshal(msgparts[i+3], &msg.ParentHeader)
	json.Unmarshal(msgparts[i+4], &msg.Metadata)
	json.Unmarshal(msgparts[i+5], &msg.Content)
	return msg, identities, nil
}

func (msg ComposedMsg) ToWireMsg(signkey []byte) ([][]byte, error) {
	msgparts := make([][]byte, 5)

	header, err := json.Marshal(msg.Header)
	if err != nil {
		return msgparts, err
	}
	msgparts[1] = header

	parentHeader, err := json.Marshal(msg.ParentHeader)
	if err != nil {
		return msgparts, err
	}
	msgparts[2] = parentHeader

	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}

	metadata, err := json.Marshal(msg.Metadata)
	if err != nil {
		return msgparts, err
	}
	msgparts[3] = metadata

	content, err := json.Marshal(msg.Content)
	if err != nil {
		return msgparts, err
	}
	msgparts[4] = content

	if len(signkey) != 0 {
		mac := hmac.New(sha256.New, signkey)
		for _, msgpart := range msgparts[1:] {
			mac.Write(msgpart)
		}
		msgparts[0] = make([]byte, hex.EncodedLen(mac.Size()))
		hex.Encode(msgparts[0], mac.Sum(nil))
	}

	return msgparts, nil
}

func NewMsg(msgType string, parent ComposedMsg) (ComposedMsg, error) {
	var msg ComposedMsg

	msg.ParentHeader = parent.Header
	msg.Header.Session = parent.Header.Session
	msg.Header.Username = parent.Header.Username
	msg.Header.MsgType = msgType
	msg.Header.ProtocolVersion = ProtocolVersion
	msg.Header.Timestamp = time.Now().UTC().Format(time.RFC3339)

	u, err := uuid.NewV4()
	if err != nil {
		return msg, err
	}
	msg.Header.MsgID = u.String()

	return msg, nil
}
