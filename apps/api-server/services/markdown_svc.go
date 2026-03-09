package services

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type MarkdownTask struct {
	TaskID     int    `json:"task_id"`
	RawLine    string `json:"raw_line"`
	Completed  bool   `json:"completed"`
	Status     string `json:"status"`
	Subject    string `json:"subject"`
	GroupTitle string `json:"group_title"`
	Content    string `json:"content"`
}

var (
	legacyTaskPattern     = regexp.MustCompile(`^\s*-\s*\[([ xX])\]\s*([^:]+):\s*(.*)$`)
	checkboxTaskPattern   = regexp.MustCompile(`^\s*-\s*\[([ xX])\]\s*(.+)$`)
	subjectHeadingPattern = regexp.MustCompile(`^###\s+(.+)$`)
	groupHeadingPattern   = regexp.MustCompile(`^####\s+(.+)$`)
)

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

func parseTaskLine(line string, currentSubject string, currentGroup string, taskID int) (MarkdownTask, bool) {
	if matches := legacyTaskPattern.FindStringSubmatch(line); len(matches) == 4 {
		return newMarkdownTask(taskID, line, matches[1], matches[2], matches[3], matches[3]), true
	}

	matches := checkboxTaskPattern.FindStringSubmatch(line)
	if len(matches) != 3 {
		return MarkdownTask{}, false
	}

	subject, groupTitle, content := normalizeStoredTask(currentSubject, currentGroup, matches[2])
	return newMarkdownTask(taskID, line, matches[1], subject, groupTitle, content), true
}

func newMarkdownTask(taskID int, rawLine string, checkedMark string, subject string, groupTitle string, content string) MarkdownTask {
	normalizedSubject, normalizedGroupTitle, normalizedContent := normalizeStoredTask(subject, groupTitle, content)
	completed := strings.EqualFold(strings.TrimSpace(checkedMark), "x")
	status := "pending"
	if completed {
		status = "completed"
	}

	return MarkdownTask{
		TaskID:     taskID,
		RawLine:    strings.TrimSpace(rawLine),
		Completed:  completed,
		Status:     status,
		Subject:    normalizedSubject,
		GroupTitle: normalizedGroupTitle,
		Content:    normalizedContent,
	}
}

func renderTaskLine(task MarkdownTask) string {
	checkMark := " "
	if task.Completed {
		checkMark = "x"
	}
	return fmt.Sprintf("- [%s] %s", checkMark, strings.TrimSpace(task.Content))
}

func renderMarkdownDocument(date time.Time, tasks []MarkdownTask) string {
	header := fmt.Sprintf("# %s - 今日成长轨迹\n\n## 🎯 任务清单\n", date.Format("2006年01月02日"))
	if len(tasks) == 0 {
		return header
	}

	type homeworkGroup struct {
		Title string
		Tasks []MarkdownTask
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

		group.Tasks = append(group.Tasks, MarkdownTask{
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

func writeTasksToMD(path string, date time.Time, tasks []MarkdownTask) error {
	return os.WriteFile(path, []byte(renderMarkdownDocument(date, tasks)), 0o644)
}

// EnsureDailyFile creates the markdown file and its directories with a default header if it doesn't exist.
func EnsureDailyFile(familyID, userID uint, date time.Time) (string, error) {
	path := getDailyFilePath(familyID, userID, date)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return "", err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := writeTasksToMD(path, date, []MarkdownTask{}); err != nil {
			return "", err
		}
	}
	return path, nil
}

func SaveTaskWithGroupToMDAtDate(familyID, userID uint, subject, groupTitle, content string, date time.Time) error {
	subject, groupTitle, content = normalizeStoredTask(subject, groupTitle, content)
	if content == "" {
		return fmt.Errorf("task content cannot be empty")
	}

	path, err := EnsureDailyFile(familyID, userID, date)
	if err != nil {
		return err
	}

	tasks, err := GetTasksFromMD(familyID, userID, date)
	if err != nil {
		return err
	}

	tasks = append(tasks, MarkdownTask{
		Subject:    subject,
		GroupTitle: groupTitle,
		Content:    content,
		Completed:  false,
		Status:     "pending",
	})

	return writeTasksToMD(path, date, tasks)
}

func SaveTaskToMDAtDate(familyID, userID uint, subject, content string, date time.Time) error {
	return SaveTaskWithGroupToMDAtDate(familyID, userID, subject, "", content, date)
}

func SaveTaskToMD(familyID, userID uint, subject, content string) error {
	return SaveTaskToMDAtDate(familyID, userID, subject, content, time.Now())
}

func GetTasksFromMD(familyID, userID uint, date time.Time) ([]MarkdownTask, error) {
	path := getDailyFilePath(familyID, userID, date)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []MarkdownTask{}, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	tasks := make([]MarkdownTask, 0)
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

func updateTaskSet(familyID, userID uint, date time.Time, matcher func(task MarkdownTask) bool, completed bool) ([]MarkdownTask, int, error) {
	path, err := EnsureDailyFile(familyID, userID, date)
	if err != nil {
		return nil, 0, err
	}

	tasks, err := GetTasksFromMD(familyID, userID, date)
	if err != nil {
		return nil, 0, err
	}

	updatedCount := 0
	for index := range tasks {
		if !matcher(tasks[index]) {
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

	if err := writeTasksToMD(path, date, tasks); err != nil {
		return nil, 0, err
	}

	updatedTasks, err := GetTasksFromMD(familyID, userID, date)
	return updatedTasks, updatedCount, err
}

func UpdateTaskCompletionByID(familyID, userID uint, date time.Time, taskID int, completed bool) ([]MarkdownTask, int, error) {
	return updateTaskSet(familyID, userID, date, func(task MarkdownTask) bool {
		return task.TaskID == taskID
	}, completed)
}

func UpdateTaskCompletionBySubject(familyID, userID uint, date time.Time, subject string, completed bool) ([]MarkdownTask, int, error) {
	normalizedSubject, _, _ := normalizeStoredTask(subject, "", "placeholder")
	return updateTaskSet(familyID, userID, date, func(task MarkdownTask) bool {
		return task.Subject == normalizedSubject
	}, completed)
}

func UpdateTaskCompletionByHomeworkGroup(familyID, userID uint, date time.Time, subject string, groupTitle string, completed bool) ([]MarkdownTask, int, error) {
	normalizedSubject, normalizedGroupTitle, _ := normalizeStoredTask(subject, groupTitle, "placeholder")
	return updateTaskSet(familyID, userID, date, func(task MarkdownTask) bool {
		return task.Subject == normalizedSubject && task.GroupTitle == normalizedGroupTitle
	}, completed)
}

func UpdateAllTasksCompletion(familyID, userID uint, date time.Time, completed bool) ([]MarkdownTask, int, error) {
	return updateTaskSet(familyID, userID, date, func(task MarkdownTask) bool {
		return true
	}, completed)
}
