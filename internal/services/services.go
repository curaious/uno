package services

import (
	"log/slog"

	"github.com/praveen001/uno/internal/config"
	"github.com/praveen001/uno/internal/db"
	agent2 "github.com/praveen001/uno/internal/services/agent"
	conversation2 "github.com/praveen001/uno/internal/services/conversation"
	mcp_server2 "github.com/praveen001/uno/internal/services/mcp_server"
	model2 "github.com/praveen001/uno/internal/services/model"
	project2 "github.com/praveen001/uno/internal/services/project"
	prompt2 "github.com/praveen001/uno/internal/services/prompt"
	provider2 "github.com/praveen001/uno/internal/services/provider"
	schema2 "github.com/praveen001/uno/internal/services/schema"
	traces2 "github.com/praveen001/uno/internal/services/traces"
	user2 "github.com/praveen001/uno/internal/services/user"
	virtual_key2 "github.com/praveen001/uno/internal/services/virtual_key"
)

type Services struct {
	Provider     *provider2.ProviderService
	Model        *model2.ModelService
	Agent        *agent2.AgentService
	MCPServer    *mcp_server2.MCPServerService
	Project      *project2.ProjectService
	Prompt       *prompt2.PromptService
	Schema       *schema2.SchemaService
	Conversation *conversation2.ConversationService
	VirtualKey   *virtual_key2.VirtualKeyService
	Traces       *traces2.TracesService
	User         *user2.UserService
}

func NewServices(conf *config.Config) *Services {
	dbconn := db.NewConn(conf)

	var tracesSvc *traces2.TracesService
	if conf.CLICKHOUSE_HOST != "" {
		chConn, err := traces2.NewClickHouseConn(&traces2.ClickHouseConfig{
			Host:     conf.CLICKHOUSE_HOST,
			Port:     conf.CLICKHOUSE_PORT,
			Database: conf.CLICKHOUSE_DATABASE,
			Username: conf.CLICKHOUSE_USERNAME,
			Password: conf.CLICKHOUSE_PASSWORD,
			UseTLS:   conf.CLICKHOUSE_USE_TLS,
		})
		if err != nil {
			slog.Warn("Failed to connect to ClickHouse for traces", slog.Any("error", err))
		} else {
			tracesSvc = traces2.NewTracesService(chConn)
			slog.Info("Connected to ClickHouse for traces")
		}
	}

	return &Services{
		Provider:     provider2.NewProviderService(provider2.NewProviderRepo(dbconn)),
		VirtualKey:   virtual_key2.NewVirtualKeyService(virtual_key2.NewVirtualKeyRepo(dbconn)),
		Project:      project2.NewProjectService(project2.NewProjectRepo(dbconn)),
		Prompt:       prompt2.NewPromptService(prompt2.NewPromptRepo(dbconn)),
		Schema:       schema2.NewSchemaService(schema2.NewSchemaRepo(dbconn)),
		Model:        model2.NewModelService(model2.NewModelRepo(dbconn)),
		Agent:        agent2.NewAgentService(agent2.NewAgentRepo(dbconn)),
		MCPServer:    mcp_server2.NewMCPServerService(mcp_server2.NewMCPServerRepo(dbconn)),
		Conversation: conversation2.NewConversationService(conversation2.NewConversationRepo(dbconn)),
		Traces:       tracesSvc,
		User:         user2.NewUserService(user2.NewUserRepo(dbconn)),
	}
}
