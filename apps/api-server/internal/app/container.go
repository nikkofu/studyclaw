package app

import (
	taskparse "github.com/nikkofu/studyclaw/api-server/internal/modules/agent/taskparse"
	weeklyinsights "github.com/nikkofu/studyclaw/api-server/internal/modules/agent/weeklyinsights"
	taskboardapp "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/application"
	taskboardmarkdown "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/infrastructure/markdown"
	"github.com/nikkofu/studyclaw/api-server/internal/platform/llm"
)

type Container struct {
	Taskboard *taskboardapp.Service
	TaskParse *taskparse.Service
	Weekly    *weeklyinsights.Service
}

func NewContainer() *Container {
	repository := taskboardmarkdown.NewRepository()
	llmClient := llm.NewOpenAICompatibleClient(nil)

	return &Container{
		Taskboard: taskboardapp.NewService(repository),
		TaskParse: taskparse.NewService(llmClient),
		Weekly:    weeklyinsights.NewService(llmClient),
	}
}
