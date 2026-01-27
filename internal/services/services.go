package services

import (
	"log/slog"
	"os"
	"path"

	"github.com/curaious/uno/internal/config"
	"github.com/curaious/uno/internal/db"
	agent_config2 "github.com/curaious/uno/internal/services/agent_config"
	conversation2 "github.com/curaious/uno/internal/services/conversation"
	project2 "github.com/curaious/uno/internal/services/project"
	prompt2 "github.com/curaious/uno/internal/services/prompt"
	provider2 "github.com/curaious/uno/internal/services/provider"
	traces2 "github.com/curaious/uno/internal/services/traces"
	user2 "github.com/curaious/uno/internal/services/user"
	virtual_key2 "github.com/curaious/uno/internal/services/virtual_key"
	"github.com/curaious/uno/pkg/sandbox"
	"github.com/curaious/uno/pkg/sandbox/docker_sandbox"
)

type Services struct {
	Provider     *provider2.ProviderService
	AgentConfig  *agent_config2.AgentConfigService
	Project      *project2.ProjectService
	Prompt       *prompt2.PromptService
	Conversation *conversation2.ConversationService
	VirtualKey   *virtual_key2.VirtualKeyService
	Traces       *traces2.TracesService
	Sandbox      sandbox.Manager
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

	svc := &Services{
		Provider:     provider2.NewProviderService(provider2.NewProviderRepo(dbconn)),
		VirtualKey:   virtual_key2.NewVirtualKeyService(virtual_key2.NewVirtualKeyRepo(dbconn)),
		Project:      project2.NewProjectService(project2.NewProjectRepo(dbconn)),
		Prompt:       prompt2.NewPromptService(prompt2.NewPromptRepo(dbconn)),
		AgentConfig:  agent_config2.NewAgentConfigService(agent_config2.NewAgentConfigRepo(dbconn)),
		Conversation: conversation2.NewConversationService(conversation2.NewConversationRepo(dbconn)),
		Traces:       tracesSvc,
		User:         user2.NewUserService(user2.NewUserRepo(dbconn)),
	}

	// Initialize sandbox manager if explicitly enabled via environment / helm values.
	if config.GetEnvOrDefault("SANDBOX_ENABLED", "false") == "true" {
		wd, err := os.Getwd()
		if err != nil {
			slog.Warn("Failed to get working directory", slog.Any("error", err))
			wd = "/"
		}

		sMgr := docker_sandbox.NewManager(docker_sandbox.Config{
			RootDir: path.Join(wd, "sandbox-data"),
		})
		svc.Sandbox = sMgr

		slog.Info("Sandbox manager initialized")
	}

	return svc
}

// Note: additional helper functions should live in dedicated files to keep
// this constructor focused on wiring services together.
