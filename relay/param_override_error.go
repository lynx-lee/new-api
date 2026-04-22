package relay

import (
	relaycommon "github.com/QuantumNous/ai-bridge/relay/common"
	"github.com/QuantumNous/ai-bridge/types"
)

func newAPIErrorFromParamOverride(err error) *types.AIBridgeError {
	if fixedErr, ok := relaycommon.AsParamOverrideReturnError(err); ok {
		return relaycommon.AIBridgeErrorFromParamOverride(fixedErr)
	}
	return types.NewError(err, types.ErrorCodeChannelParamOverrideInvalid, types.ErrOptionWithSkipRetry())
}
