package markdown

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
	taskMetadataPattern   = regexp.MustCompile(`^\s*<!--\s*studyclaw:task:(\{.*\})\s*-->\s*$`)
)

type taskMetadata struct {
	TaskType               string `json:"task_type,omitempty"`
	ReferenceTitle         string `json:"reference_title,omitempty"`
	ReferenceAuthor        string `json:"reference_author,omitempty"`
	ReferenceText          string `json:"reference_text,omitempty"`
	ReferenceSource        string `json:"reference_source,omitempty"`
	HideReferenceFromChild bool   `json:"hide_reference_from_child,omitempty"`
	AnalysisMode           string `json:"analysis_mode,omitempty"`
}

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

func normalizeReferenceSourceValue(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", "manual", "extracted", "llm":
		return normalized
	default:
		return normalized
	}
}

func normalizeTaskMetadataFields(taskType, referenceTitle, referenceAuthor, referenceText, referenceSource string, hideReferenceFromChild bool, analysisMode string) taskMetadata {
	metadata := taskMetadata{
		TaskType:               strings.TrimSpace(taskType),
		ReferenceTitle:         strings.TrimSpace(referenceTitle),
		ReferenceAuthor:        strings.TrimSpace(referenceAuthor),
		ReferenceText:          strings.TrimSpace(referenceText),
		ReferenceSource:        normalizeReferenceSourceValue(referenceSource),
		HideReferenceFromChild: hideReferenceFromChild,
		AnalysisMode:           strings.TrimSpace(analysisMode),
	}
	if metadata.ReferenceText == "" {
		metadata.HideReferenceFromChild = false
	}
	if metadata.ReferenceTitle == "" &&
		metadata.ReferenceAuthor == "" &&
		metadata.ReferenceText == "" &&
		metadata.AnalysisMode == "" &&
		!metadata.HideReferenceFromChild {
		metadata.ReferenceSource = ""
	}
	return metadata
}

func applyTaskMetadata(task domain.Task, metadata taskMetadata) domain.Task {
	normalized := normalizeTaskMetadataFields(
		metadata.TaskType,
		metadata.ReferenceTitle,
		metadata.ReferenceAuthor,
		metadata.ReferenceText,
		metadata.ReferenceSource,
		metadata.HideReferenceFromChild,
		metadata.AnalysisMode,
	)
	task.TaskType = normalized.TaskType
	task.ReferenceTitle = normalized.ReferenceTitle
	task.ReferenceAuthor = normalized.ReferenceAuthor
	task.ReferenceText = normalized.ReferenceText
	task.ReferenceSource = normalized.ReferenceSource
	task.HideReferenceFromChild = normalized.HideReferenceFromChild
	task.AnalysisMode = normalized.AnalysisMode
	return task
}

func taskMetadataFromTask(task domain.Task) (taskMetadata, bool) {
	metadata := normalizeTaskMetadataFields(
		task.TaskType,
		task.ReferenceTitle,
		task.ReferenceAuthor,
		task.ReferenceText,
		task.ReferenceSource,
		task.HideReferenceFromChild,
		task.AnalysisMode,
	)
	if metadata.TaskType == "" &&
		metadata.ReferenceTitle == "" &&
		metadata.ReferenceAuthor == "" &&
		metadata.ReferenceText == "" &&
		metadata.ReferenceSource == "" &&
		!metadata.HideReferenceFromChild &&
		metadata.AnalysisMode == "" {
		return taskMetadata{}, false
	}
	return metadata, true
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

func renderTaskMetadataLine(task domain.Task) string {
	metadata, ok := taskMetadataFromTask(task)
	if !ok {
		return ""
	}
	content, err := json.Marshal(metadata)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("  <!-- studyclaw:task:%s -->", string(content))
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
			Completed:              task.Completed,
			Status:                 task.Status,
			Subject:                subject,
			GroupTitle:             groupTitle,
			Content:                content,
			TaskType:               task.TaskType,
			ReferenceTitle:         task.ReferenceTitle,
			ReferenceAuthor:        task.ReferenceAuthor,
			ReferenceText:          task.ReferenceText,
			ReferenceSource:        task.ReferenceSource,
			HideReferenceFromChild: task.HideReferenceFromChild,
			AnalysisMode:           task.AnalysisMode,
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
				if metadataLine := renderTaskMetadataLine(task); metadataLine != "" {
					builder.WriteString(metadataLine)
					builder.WriteString("\n")
				}
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

func (r *Repository) AddTask(input domain.CreateTaskInput, date time.Time) error {
	subject, groupTitle, content := normalizeStoredTask(input.Subject, input.GroupTitle, input.Content)
	if content == "" {
		return fmt.Errorf("task content cannot be empty")
	}

	path, err := r.EnsureDailyFile(input.FamilyID, input.AssigneeID, date)
	if err != nil {
		return err
	}

	tasks, err := r.GetTasks(input.FamilyID, input.AssigneeID, date)
	if err != nil {
		return err
	}

	task := applyTaskMetadata(domain.Task{
		Subject:    subject,
		GroupTitle: groupTitle,
		Content:    content,
		Completed:  false,
		Status:     "pending",
	}, taskMetadata{
		TaskType:               input.TaskType,
		ReferenceTitle:         input.ReferenceTitle,
		ReferenceAuthor:        input.ReferenceAuthor,
		ReferenceText:          input.ReferenceText,
		ReferenceSource:        input.ReferenceSource,
		HideReferenceFromChild: input.HideReferenceFromChild,
		AnalysisMode:           input.AnalysisMode,
	})
	tasks = append(tasks, task)

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

		if matches := taskMetadataPattern.FindStringSubmatch(line); len(matches) == 2 {
			if len(tasks) == 0 {
				continue
			}
			var metadata taskMetadata
			if err := json.Unmarshal([]byte(matches[1]), &metadata); err != nil {
				continue
			}
			tasks[len(tasks)-1] = applyTaskMetadata(tasks[len(tasks)-1], metadata)
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

func (r *Repository) ReplaceTasks(familyID, userID uint, date time.Time, tasks []domain.Task) error {
	path, err := r.EnsureDailyFile(familyID, userID, date)
	if err != nil {
		return err
	}
	return writeTasksToMD(path, date, tasks)
}

func (r *Repository) ListAvailableDates(familyID, userID uint) ([]time.Time, error) {
	dir := getWorkspacePath(familyID, userID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []time.Time{}, nil
		}
		return nil, err
	}

	dates := make([]time.Time, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		date, err := time.Parse("2006-01-02", strings.TrimSuffix(entry.Name(), ".md"))
		if err != nil {
			continue
		}
		dates = append(dates, date)
	}

	sort.Slice(dates, func(i, j int) bool {
		return dates[i].Before(dates[j])
	})
	return dates, nil
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
