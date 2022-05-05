package signedmsg

import (
	"encoding/hex"
	"testing"

	"github.com/herumi/bls-eth-go-binary/bls"
	"github.com/stretchr/testify/require"

	forksprotocol "github.com/bloxapp/ssv/protocol/forks"
	"github.com/bloxapp/ssv/protocol/v1/blockchain/beacon"
	"github.com/bloxapp/ssv/protocol/v1/message"
	"github.com/bloxapp/ssv/utils/threshold"
)

func _byteArray(input string) []byte {
	res, _ := hex.DecodeString(input)
	return res
}

// GenerateNodes generates randomly nodes
func GenerateNodes(cnt int) (map[uint64]*bls.SecretKey, map[message.OperatorID]*beacon.Node) {
	threshold.Init()
	nodes := make(map[message.OperatorID]*beacon.Node)
	sks := make(map[uint64]*bls.SecretKey)
	for i := 1; i <= cnt; i++ {
		sk := &bls.SecretKey{}
		sk.SetByCSPRNG()

		nodes[message.OperatorID(i)] = &beacon.Node{
			IbftID: uint64(i),
			Pk:     sk.GetPublicKey().Serialize(),
		}
		sks[uint64(i)] = sk
	}
	return sks, nodes
}

// SignMsg signs the given message by the given private key
func SignMsg(t *testing.T, ids []message.OperatorID, sks []*bls.SecretKey, msg *message.ConsensusMessage) *message.SignedMessage {
	threshold.Init()

	var agg *bls.Sign
	for _, sk := range sks {
		signature, err := msg.Sign(sk, forksprotocol.V1ForkVersion.String())
		require.NoError(t, err)
		if agg == nil {
			agg = signature
		} else {
			agg.Add(signature)
		}
	}

	require.NotNil(t, agg)
	return &message.SignedMessage{
		Message:   msg,
		Signature: agg.Serialize(),
		Signers:   ids,
	}
}

func TestAuthorizeMsg(t *testing.T) {
	threshold.Init()
	sks, committee := GenerateNodes(4)
	tests := []struct {
		name          string
		msg           *message.ConsensusMessage
		signerIds     []message.OperatorID
		sks           []*bls.SecretKey
		sig           []byte
		expectedError string
	}{
		{
			"valid sig",
			&message.ConsensusMessage{
				MsgType:    message.DecidedMsgType,
				Round:      4,
				Identifier: []byte{1, 2, 3, 4},
				Height:     1,
				Data:       []byte("hello"),
			},
			[]message.OperatorID{1},
			[]*bls.SecretKey{sks[1]},
			nil,
			"",
		},
		{
			"valid aggregate sig",
			&message.ConsensusMessage{
				MsgType:    message.DecidedMsgType,
				Round:      4,
				Identifier: []byte{1, 2, 3, 4},
				Height:     1,
				Data:       []byte("hello"),
			},
			[]message.OperatorID{1, 2},
			[]*bls.SecretKey{sks[1], sks[2]},
			nil,
			"",
		},
		{
			"invalid sig",
			&message.ConsensusMessage{
				MsgType:    message.DecidedMsgType,
				Round:      4,
				Identifier: []byte{1, 2, 3, 4},
				Height:     1,
				Data:       []byte("hello"),
			},
			[]message.OperatorID{1},
			[]*bls.SecretKey{sks[1]},
			_byteArray("b4fa352d2d6dbdf884266af7ea0914451929b343527ea6c1737ac93b3dde8b7c98e6ce61d68b7a2e7b7af8f8d0fd429d0bdd5f930b83e6842bf4342d3d1d3d10fc0d15bab7649bb8aa8287ca104a1f79d396ce0217bb5cd3e6503a3bce4c9776"),
			"failed to verify signature",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			signed := SignMsg(t, test.signerIds, test.sks, test.msg)
			if test.sig != nil {
				signed.Signature = test.sig
			}

			pipeline := AuthorizeMsg(&beacon.Share{
				Committee: committee,
			}, forksprotocol.V1ForkVersion.String())

			if len(test.expectedError) == 0 {
				require.NoError(t, pipeline.Run(signed))
			} else {
				require.EqualError(t, pipeline.Run(signed), test.expectedError)
			}
		})
	}
}
