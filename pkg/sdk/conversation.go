package sdk

import (
	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/agent-framework/history"
	"github.com/praveen001/uno/pkg/sdk/adapters"
)

func (c *SDK) NewConversationManager(namespace, msgId, previousMsgId string, opts ...history.ConversationManagerOptions) core.ChatHistory {
	return history.NewConversationManager(
		c.getConversationPersistence(),
		c.projectId,
		namespace,
		msgId,
		previousMsgId,
		opts...,
	)
}

func (c *SDK) getConversationPersistence() history.ConversationPersistenceManager {
	if c.directMode {
		return nil
	}

	return adapters.NewExternalConversationPersistence(c.endpoint)
}
