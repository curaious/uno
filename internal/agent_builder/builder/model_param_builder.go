package builder

import (
	"github.com/bytedance/sonic"
	"github.com/curaious/uno/internal/services/agent_config"
	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/llm/responses"
)

func BuildModelParams(params *agent_config.ModelConfig) (responses.Parameters, error) {
	var modelParams responses.Parameters
	buf, err := sonic.Marshal(params)
	if err != nil {
		return modelParams, err
	}

	if err = sonic.Unmarshal(buf, &modelParams); err != nil {
		return modelParams, err
	}

	if modelParams.Reasoning != nil {
		modelParams.Include = []responses.Includable{responses.IncludableReasoningEncryptedContent}
		modelParams.Reasoning.Summary = utils.Ptr("auto")
	}

	return modelParams, nil
}
