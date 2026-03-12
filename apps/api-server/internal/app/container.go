package app

import (
	taskparse "github.com/nikkofu/studyclaw/api-server/internal/modules/agent/taskparse"
	wordparse "github.com/nikkofu/studyclaw/api-server/internal/modules/agent/wordparse"
	weeklyinsights "github.com/nikkofu/studyclaw/api-server/internal/modules/agent/weeklyinsights"
	taskboardapp "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/application"
	taskboardjson "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/infrastructure/jsonstore"
	taskboardmarkdown "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/infrastructure/markdown"
	"github.com/nikkofu/studyclaw/api-server/internal/platform/llm"
)

type Container struct {
	Taskboard *taskboardapp.Service
	PhaseOne  *taskboardapp.PhaseOneService
	TaskParse *taskparse.Service
	WordParse *wordparse.Service
	Weekly    *weeklyinsights.Service
}

func NewContainer() *Container {
	repository := taskboardmarkdown.NewRepository()
	phaseOneRepo := taskboardjson.NewRepository()
	llmClient := llm.NewOpenAICompatibleClient(nil)
	taskboardService := taskboardapp.NewService(repository)

	return &Container{
		Taskboard: taskboardService,
		PhaseOne:  taskboardapp.NewPhaseOneService(taskboardService, phaseOneRepo),
		TaskParse: taskparse.NewService(llmClient),
		WordParse: wordparse.NewService(llmClient),
		Weekly:    weeklyinsights.NewService(llmClient),
	}
}
