package sdk

import (
	"github.com/praveen001/uno/pkg/agent-framework/history"
	"github.com/praveen001/uno/pkg/sdk/adapters"
)

func (c *SDK) NewConversationManager(opts ...history.ConversationManagerOptions) *history.CommonConversationManager {
	return history.NewConversationManager(
		c.getConversationPersistence(),
		opts...,
	)
}

func (c *SDK) getConversationPersistence() history.ConversationPersistenceAdapter {
	if c.endpoint == "" {
		return adapters.NewInMemoryConversationPersistence()
	}

	return adapters.NewExternalConversationPersistence(c.endpoint, c.projectId)
}
