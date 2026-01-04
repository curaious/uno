package sdk

import (
	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/agent-framework/history"
	"github.com/praveen001/uno/pkg/sdk/adapters"
)

func (c *SDK) NewConversationManager(opts ...history.ConversationManagerOptions) core.ChatHistory {
	return history.NewConversationManager(
		c.getConversationPersistence(),
		opts...,
	)
}

func (c *SDK) getConversationPersistence() history.ConversationPersistenceAdapter {
	if c.endpoint == "" {
		return nil
	}

	return adapters.NewExternalConversationPersistence(c.endpoint, c.projectId)
}
