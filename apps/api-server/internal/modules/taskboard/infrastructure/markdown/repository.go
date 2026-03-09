package markdown

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/domain"
)

type Repository struct{}

var (
	legacyTaskPattern     = regexp.MustCompile(`^\s*-\s*\[([ xX])\]\s*([^:]+):\s*(.*)$`)
	checkboxTaskPattern   = regexp.MustCompile(`^\s*-\s*\[([ xX])\]\s*(.+)$`)
	subjectHeadingPattern = regexp.MustCompile(`^###\s+(.+)$`)
	groupHeadingPattern   = regexp.MustCompile(`^####\s+(.+)$`)
)

func NewRepository() *Repository {
	return &Repository{}
}

func getDataRoot() string {
	if root := os.Getenv("STUDYCLAW_DATA_DIR"); root != "" {
		return root
	}

	cwd, _ := os.Getwd()
	return filepath.Join(cwd, "..", "..", "data")
}

func getWorkspacePath(familyID, userID uint) string {
	return filepath.Join(getDataRoot(), "workspaces", fmt.Sprintf("family_%d", familyID), fmt.Sprintf("user_%d", userID))
}

func getDailyFilePath(familyID, userID uint, date time.Time) string {
	dir := getWorkspacePath(familyID, userID)
	filename := date.Format("2006-01-02") + ".md"
	return filepath.Join(dir, filename)
}

func normalizeStoredTask(subject, groupTitle, content string) (string, string, string) {
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

func newMarkdownTask(taskID int, rawLine string, checkedMark string, subject string, groupTitle string, content string) domain.Task {
	normalizedSubject, normalizedGroupTitle, normalizedContent := normalizeStoredTask(subject, groupTitle, content)
	completed := strings.EqualFold(strings.TrimSpace(checkedMark), "x")
	status := "pending"
	if completed {
		status = "completed"
	}

	return domain.Task{
		TaskID:     taskID,
		RawLine:    strings.TrimSpace(rawLine),
		Completed:  completed,
		Status:     status,
		Subject:    normalizedSubject,
		GroupTitle: normalizedGroupTitle,
		Content:    normalizedContent,
	}
}

func parseTaskLine(line string, currentSubject string, currentGroup string, taskID int) (domain.Task, bool) {
	if matches := legacyTaskPattern.FindStringSubmatch(line); len(matches) == 4 {
		return newMarkdownTask(taskID, line, matches[1], matches[2], matches[3], matches[3]), true
	}

	matches := checkboxTaskPattern.FindStringSubmatch(line)
	if len(matches) != 3 {
		return domain.Task{}, false
	}

	subject, groupTitle, content := normalizeStoredTask(currentSubject, currentGroup, matches[2])
	return newMarkdownTask(taskID, line, matches[1], subject, groupTitle, content), true
}

func renderTaskLine(task domain.Task) string {
	checkMark := " "
	if task.Completed {
		checkMark = "x"
	}
	return fmt.Sprintf("- [%s] %s", checkMark, strings.TrimSpace(task.Content))
}

func renderMarkdownDocument(date time.Time, tasks []domain.Task) string {
	header := fmt.Sprintf("# %s - 今日成长轨迹\n\n## 🎯 任务清单\n", date.Format("2006年01月02日"))
	if len(tasks) == 0 {
		return header
	}

	type homeworkGroup struct {
		Title string
		Tasks []domain.Task
	}

	type subjectBucket struct {
		Subject string
		Groups  []*homeworkGroup
	}

	subjectBuckets := make(map[string]*subjectBucket)
	subjectOrder := make([]string, 0)
	groupOrders := make(map[string][]string)
	groupBuckets := make(map[string]map[string]*homeworkGroup)

	for _, task := range tasks {
		subject, groupTitle, content := normalizeStoredTask(task.Subject, task.GroupTitle, task.Content)
		if content == "" {
			continue
		}

		bucket, exists := subjectBuckets[subject]
		if !exists {
			bucket = &subjectBucket{Subject: subject}
			subjectBuckets[subject] = bucket
			subjectOrder = append(subjectOrder, subject)
			groupBuckets[subject] = make(map[string]*homeworkGroup)
		}

		group, exists := groupBuckets[subject][groupTitle]
		if !exists {
			group = &homeworkGroup{Title: groupTitle}
			groupBuckets[subject][groupTitle] = group
			groupOrders[subject] = append(groupOrders[subject], groupTitle)
			bucket.Groups = append(bucket.Groups, group)
		}

		group.Tasks = append(group.Tasks, domain.Task{
			Completed:  task.Completed,
			Status:     task.Status,
			Subject:    subject,
			GroupTitle: groupTitle,
			Content:    content,
		})
	}

	var builder strings.Builder
	builder.WriteString(header)

	for _, subject := range subjectOrder {
		builder.WriteString(fmt.Sprintf("\n### %s\n", subject))

		for _, groupTitle := range groupOrders[subject] {
			group := groupBuckets[subject][groupTitle]
			builder.WriteString(fmt.Sprintf("\n#### %s\n", group.Title))
			for _, task := range group.Tasks {
				builder.WriteString(renderTaskLine(task))
				builder.WriteString("\n")
			}
		}
	}

	return builder.String()
}

func writeTasksToMD(path string, date time.Time, tasks []domain.Task) error {
	return os.WriteFile(path, []byte(renderMarkdownDocument(date, tasks)), 0o644)
}

func (r *Repository) EnsureDailyFile(familyID, userID uint, date time.Time) (string, error) {
	path := getDailyFilePath(familyID, userID, date)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return "", err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := writeTasksToMD(path, date, []domain.Task{}); err != nil {
			return "", err
		}
	}
	return path, nil
}

func (r *Repository) AddTask(familyID, userID uint, subject, groupTitle, content string, date time.Time) error {
	subject, groupTitle, content = normalizeStoredTask(subject, groupTitle, content)
	if content == "" {
		return fmt.Errorf("task content cannot be empty")
	}

	path, err := r.EnsureDailyFile(familyID, userID, date)
	if err != nil {
		return err
	}

	tasks, err := r.GetTasks(familyID, userID, date)
	if err != nil {
		return err
	}

	tasks = append(tasks, domain.Task{
		Subject:    subject,
		GroupTitle: groupTitle,
		Content:    content,
		Completed:  false,
		Status:     "pending",
	})

	return writeTasksToMD(path, date, tasks)
}

func (r *Repository) GetTasks(familyID, userID uint, date time.Time) ([]domain.Task, error) {
	path := getDailyFilePath(familyID, userID, date)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []domain.Task{}, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	tasks := make([]domain.Task, 0)
	scanner := bufio.NewScanner(file)
	taskID := 0
	currentSubject := ""
	currentGroup := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if matches := subjectHeadingPattern.FindStringSubmatch(line); len(matches) == 2 {
			currentSubject = strings.TrimSpace(matches[1])
			currentGroup = ""
			continue
		}

		if matches := groupHeadingPattern.FindStringSubmatch(line); len(matches) == 2 {
			currentGroup = strings.TrimSpace(matches[1])
			continue
		}

		if strings.HasPrefix(line, "#") {
			continue
		}

		task, ok := parseTaskLine(line, currentSubject, currentGroup, taskID+1)
		if !ok {
			continue
		}

		taskID++
		tasks = append(tasks, task)
	}

	return tasks, scanner.Err()
}

func (r *Repository) updateTaskSet(familyID, userID uint, date time.Time, matcher func(task domain.Task) bool, completed bool) ([]domain.Task, int, int, error) {
	tasks, err := r.GetTasks(familyID, userID, date)
	if err != nil {
		return nil, 0, 0, err
	}

	matchedCount := 0
	updatedCount := 0
	for index := range tasks {
		if !matcher(tasks[index]) {
			continue
		}
		matchedCount++
		if tasks[index].Completed == completed {
			continue
		}
		tasks[index].Completed = completed
		if completed {
			tasks[index].Status = "completed"
		} else {
			tasks[index].Status = "pending"
		}
		updatedCount++
	}

	if updatedCount > 0 {
		path, err := r.EnsureDailyFile(familyID, userID, date)
		if err != nil {
			return nil, 0, 0, err
		}
		if err := writeTasksToMD(path, date, tasks); err != nil {
			return nil, 0, 0, err
		}
	}

	return tasks, matchedCount, updatedCount, nil
}

func (r *Repository) UpdateTaskCompletionByID(familyID, userID uint, date time.Time, taskID int, completed bool) ([]domain.Task, int, int, error) {
	return r.updateTaskSet(familyID, userID, date, func(task domain.Task) bool {
		return task.TaskID == taskID
	}, completed)
}

func (r *Repository) UpdateTaskCompletionBySubject(familyID, userID uint, date time.Time, subject string, completed bool) ([]domain.Task, int, int, error) {
	normalizedSubject, _, _ := normalizeStoredTask(subject, "", "placeholder")
	return r.updateTaskSet(familyID, userID, date, func(task domain.Task) bool {
		return task.Subject == normalizedSubject
	}, completed)
}

func (r *Repository) UpdateTaskCompletionByHomeworkGroup(familyID, userID uint, date time.Time, subject string, groupTitle string, completed bool) ([]domain.Task, int, int, error) {
	normalizedSubject, normalizedGroupTitle, _ := normalizeStoredTask(subject, groupTitle, "placeholder")
	return r.updateTaskSet(familyID, userID, date, func(task domain.Task) bool {
		return task.Subject == normalizedSubject && task.GroupTitle == normalizedGroupTitle
	}, completed)
}

func (r *Repository) UpdateAllTasksCompletion(familyID, userID uint, date time.Time, completed bool) ([]domain.Task, int, int, error) {
	return r.updateTaskSet(familyID, userID, date, func(task domain.Task) bool {
		return true
	}, completed)
}
