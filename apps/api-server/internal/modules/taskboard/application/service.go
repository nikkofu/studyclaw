package application

import (
	"fmt"
	"strings"
	"time"

	"github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/domain"
)

type Repository interface {
	EnsureDailyFile(familyID, userID uint, date time.Time) (string, error)
	AddTask(familyID, userID uint, subject, groupTitle, content string, date time.Time) error
	GetTasks(familyID, userID uint, date time.Time) ([]domain.Task, error)
	UpdateTaskCompletionByID(familyID, userID uint, date time.Time, taskID int, completed bool) ([]domain.Task, int, error)
	UpdateTaskCompletionBySubject(familyID, userID uint, date time.Time, subject string, completed bool) ([]domain.Task, int, error)
	UpdateTaskCompletionByHomeworkGroup(familyID, userID uint, date time.Time, subject string, groupTitle string, completed bool) ([]domain.Task, int, error)
	UpdateAllTasksCompletion(familyID, userID uint, date time.Time, completed bool) ([]domain.Task, int, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func ParseAssignedDate(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Now(), nil
	}

	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("assigned_date must be in YYYY-MM-DD format")
	}

	return parsed, nil
}

func NormalizeTaskFields(subject, groupTitle, content string) (string, string, string) {
	normalizedSubject := strings.TrimSpace(subject)
	if normalizedSubject == "" {
		normalizedSubject = "未分类"
	}

	normalizedContent := strings.TrimSpace(content)
	normalizedGroupTitle := strings.TrimSpace(groupTitle)
	if normalizedGroupTitle == "" {
		normalizedGroupTitle = normalizedContent
	}

	return normalizedSubject, normalizedGroupTitle, normalizedContent
}

func (s *Service) CreateTask(input domain.CreateTaskInput) (time.Time, error) {
	input.Subject, input.GroupTitle, input.Content = NormalizeTaskFields(input.Subject, input.GroupTitle, input.Content)
	if input.Content == "" {
		return time.Time{}, fmt.Errorf("content cannot be empty")
	}

	assignedDate, err := ParseAssignedDate(input.AssignedDate)
	if err != nil {
		return time.Time{}, err
	}

	if err := s.repo.AddTask(input.FamilyID, input.AssigneeID, input.Subject, input.GroupTitle, input.Content, assignedDate); err != nil {
		return time.Time{}, err
	}

	return assignedDate, nil
}

func (s *Service) CreateTasks(inputs []domain.CreateTaskInput) (time.Time, error) {
	var assignedDate time.Time
	for index, input := range inputs {
		currentDate, err := s.CreateTask(input)
		if err != nil {
			return time.Time{}, err
		}
		if index == 0 {
			assignedDate = currentDate
		}
	}

	return assignedDate, nil
}

func (s *Service) ListTasks(familyID, userID uint, date time.Time) ([]domain.Task, error) {
	return s.repo.GetTasks(familyID, userID, date)
}

func BuildBoard(date time.Time, tasks []domain.Task) domain.Board {
	groupMap := make(map[string]*domain.GroupSummary)
	homeworkMap := make(map[string]*domain.HomeworkGroupSummary)
	subjectOrder := make([]string, 0)
	homeworkOrder := make([]string, 0)
	summary := domain.Summary{
		Total:     len(tasks),
		Completed: 0,
		Pending:   0,
		Status:    "empty",
	}

	for _, task := range tasks {
		subject := strings.TrimSpace(task.Subject)
		if subject == "" {
			subject = "未分类"
		}
		groupTitle := strings.TrimSpace(task.GroupTitle)
		if groupTitle == "" {
			groupTitle = strings.TrimSpace(task.Content)
		}

		subjectGroup, exists := groupMap[subject]
		if !exists {
			subjectGroup = &domain.GroupSummary{Subject: subject}
			groupMap[subject] = subjectGroup
			subjectOrder = append(subjectOrder, subject)
		}

		homeworkKey := subject + "\x00" + groupTitle
		homeworkGroup, exists := homeworkMap[homeworkKey]
		if !exists {
			homeworkGroup = &domain.HomeworkGroupSummary{
				Subject:    subject,
				GroupTitle: groupTitle,
			}
			homeworkMap[homeworkKey] = homeworkGroup
			homeworkOrder = append(homeworkOrder, homeworkKey)
		}

		subjectGroup.Total++
		homeworkGroup.Total++
		if task.Completed {
			subjectGroup.Completed++
			homeworkGroup.Completed++
			summary.Completed++
		} else {
			subjectGroup.Pending++
			homeworkGroup.Pending++
			summary.Pending++
		}
	}

	groups := make([]domain.GroupSummary, 0, len(groupMap))
	for _, subject := range subjectOrder {
		group := groupMap[subject]
		switch {
		case group.Completed == 0:
			group.Status = "pending"
		case group.Completed == group.Total:
			group.Status = "completed"
		default:
			group.Status = "partial"
		}
		groups = append(groups, *group)
	}

	homeworkGroups := make([]domain.HomeworkGroupSummary, 0, len(homeworkMap))
	for _, homeworkKey := range homeworkOrder {
		group := homeworkMap[homeworkKey]
		switch {
		case group.Completed == 0:
			group.Status = "pending"
		case group.Completed == group.Total:
			group.Status = "completed"
		default:
			group.Status = "partial"
		}
		homeworkGroups = append(homeworkGroups, *group)
	}

	switch {
	case len(tasks) == 0:
		summary.Status = "empty"
	case summary.Completed == 0:
		summary.Status = "pending"
	case summary.Completed == len(tasks):
		summary.Status = "completed"
	default:
		summary.Status = "partial"
	}

	return domain.Board{
		Date:           date.Format("2006-01-02"),
		Tasks:          tasks,
		Groups:         groups,
		HomeworkGroups: homeworkGroups,
		Summary:        summary,
	}
}

func (s *Service) ListBoard(familyID, userID uint, date time.Time) (domain.Board, error) {
	tasks, err := s.ListTasks(familyID, userID, date)
	if err != nil {
		return domain.Board{}, err
	}

	return BuildBoard(date, tasks), nil
}

func (s *Service) UpdateTaskStatusByID(familyID, userID uint, date time.Time, taskID int, completed bool) (domain.Board, int, error) {
	tasks, updatedCount, err := s.repo.UpdateTaskCompletionByID(familyID, userID, date, taskID, completed)
	if err != nil {
		return domain.Board{}, 0, err
	}

	return BuildBoard(date, tasks), updatedCount, nil
}

func (s *Service) UpdateTaskStatusByGroup(familyID, userID uint, date time.Time, subject string, groupTitle string, completed bool) (domain.Board, int, error) {
	var (
		tasks        []domain.Task
		updatedCount int
		err          error
	)

	if strings.TrimSpace(groupTitle) != "" {
		tasks, updatedCount, err = s.repo.UpdateTaskCompletionByHomeworkGroup(familyID, userID, date, subject, groupTitle, completed)
	} else {
		tasks, updatedCount, err = s.repo.UpdateTaskCompletionBySubject(familyID, userID, date, subject, completed)
	}
	if err != nil {
		return domain.Board{}, 0, err
	}

	return BuildBoard(date, tasks), updatedCount, nil
}

func (s *Service) UpdateAllTaskStatuses(familyID, userID uint, date time.Time, completed bool) (domain.Board, int, error) {
	tasks, updatedCount, err := s.repo.UpdateAllTasksCompletion(familyID, userID, date, completed)
	if err != nil {
		return domain.Board{}, 0, err
	}

	return BuildBoard(date, tasks), updatedCount, nil
}
