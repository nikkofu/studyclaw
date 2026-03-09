import 'package:flutter/material.dart';
import 'package:pad_app/task_board/controller.dart';
import 'package:pad_app/task_board/models.dart';
import 'package:pad_app/task_board/repository.dart';

const String defaultApiBaseUrl = String.fromEnvironment(
  'API_BASE_URL',
  defaultValue: 'http://localhost:8080',
);

class PadTaskBoardPage extends StatefulWidget {
  const PadTaskBoardPage({
    super.key,
    this.autoLoad = true,
    this.initialDate,
    this.initialApiBaseUrl,
    this.initialFamilyId,
    this.initialUserId,
    this.repository = const RemoteTaskBoardRepository(),
  });

  final bool autoLoad;
  final String? initialDate;
  final String? initialApiBaseUrl;
  final int? initialFamilyId;
  final int? initialUserId;
  final TaskBoardRepository repository;

  @override
  State<PadTaskBoardPage> createState() => _PadTaskBoardPageState();
}

class _PadTaskBoardPageState extends State<PadTaskBoardPage> {
  late final TextEditingController _apiBaseUrlController;
  late final TextEditingController _familyIdController;
  late final TextEditingController _userIdController;
  late final TextEditingController _dateController;
  late final TaskBoardController _controller;

  @override
  void initState() {
    super.initState();
    _apiBaseUrlController = TextEditingController(
      text: widget.initialApiBaseUrl ?? defaultApiBaseUrl,
    );
    _familyIdController = TextEditingController(
      text: '${widget.initialFamilyId ?? 306}',
    );
    _userIdController = TextEditingController(
      text: '${widget.initialUserId ?? 1}',
    );
    _dateController = TextEditingController(
      text: widget.initialDate ?? formatTaskBoardDate(DateTime.now()),
    );
    _controller = TaskBoardController(repository: widget.repository);

    if (widget.autoLoad) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _loadBoard(showLoadingState: true);
      });
    }
  }

  @override
  void dispose() {
    _apiBaseUrlController.dispose();
    _familyIdController.dispose();
    _userIdController.dispose();
    _dateController.dispose();
    _controller.dispose();
    super.dispose();
  }

  Future<void> _loadBoard({
    bool showLoadingState = false,
    String? successMessage,
  }) async {
    final request = _buildRequest();
    if (request == null) {
      return;
    }

    await _controller.loadBoard(
      request,
      showLoadingState: showLoadingState,
      successMessage: successMessage,
    );
  }

  Future<void> _refreshBoard() async {
    final request = _buildRequest();
    if (request == null) {
      return;
    }

    await _controller.refresh(request);
  }

  Future<void> _shiftDate(int offsetInDays) async {
    final currentDate = parseTaskBoardDate(_dateController.text.trim());
    if (currentDate == null) {
      _controller.presentValidationError('日期需要是 YYYY-MM-DD。');
      return;
    }

    final nextDate = currentDate.add(Duration(days: offsetInDays));
    _dateController.text = formatTaskBoardDate(nextDate);
    await _loadBoard(successMessage: '已切换到 ${_dateController.text}');
  }

  Future<void> _pickDate() async {
    final initialDate =
        parseTaskBoardDate(_dateController.text.trim()) ?? DateTime.now();
    final selectedDate = await showDatePicker(
      context: context,
      initialDate: initialDate,
      firstDate: DateTime(initialDate.year - 5),
      lastDate: DateTime(initialDate.year + 5),
    );

    if (selectedDate == null) {
      return;
    }

    _dateController.text = formatTaskBoardDate(selectedDate);
    if (!mounted) {
      return;
    }
    await _loadBoard(successMessage: '已切换到 ${_dateController.text}');
  }

  Future<void> _updateSingleTask(TaskItem task, bool completed) async {
    final request = _buildRequest();
    if (request == null) {
      return;
    }

    await _controller.updateSingleTask(request, task, completed);
  }

  Future<void> _updateSubjectGroup(TaskGroup group, bool completed) async {
    final request = _buildRequest();
    if (request == null) {
      return;
    }

    await _controller.updateSubjectGroup(request, group, completed);
  }

  Future<void> _updateHomeworkGroup(
    HomeworkGroup group,
    bool completed,
  ) async {
    final request = _buildRequest();
    if (request == null) {
      return;
    }

    await _controller.updateHomeworkGroup(request, group, completed);
  }

  Future<void> _updateAllTasks(bool completed) async {
    final request = _buildRequest();
    if (request == null) {
      return;
    }

    await _controller.updateAllTasks(request, completed: completed);
  }

  TaskBoardRequest? _buildRequest() {
    final request = TaskBoardRequest(
      apiBaseUrl: _apiBaseUrlController.text.trim(),
      familyId: int.tryParse(_familyIdController.text.trim()) ?? 0,
      userId: int.tryParse(_userIdController.text.trim()) ?? 0,
      date: _dateController.text.trim(),
    );
    final validationError = request.validate();
    if (validationError != null) {
      _controller.presentValidationError(validationError);
      return null;
    }
    return request;
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _controller,
      builder: (context, _) {
        final state = _controller.state;

        return Scaffold(
          appBar: AppBar(
            title: const Text('孩子任务同步台'),
            actions: [
              IconButton(
                tooltip: '手动刷新',
                onPressed: state.isBusy
                    ? null
                    : () {
                        _refreshBoard();
                      },
                icon: const Icon(Icons.refresh),
              ),
            ],
          ),
          body: SafeArea(
            child: RefreshIndicator(
              onRefresh: _refreshBoard,
              child: ListView(
                physics: const AlwaysScrollableScrollPhysics(),
                padding: const EdgeInsets.all(16),
                children: [
                  _ConfigPanel(
                    apiBaseUrlController: _apiBaseUrlController,
                    familyIdController: _familyIdController,
                    userIdController: _userIdController,
                    dateController: _dateController,
                    busy: state.isBusy,
                    hasLoadedOnce: state.hasLoadedOnce,
                    lastSyncedAt: state.lastSyncedAt,
                    onLoadBoard: () {
                      _loadBoard(showLoadingState: true);
                    },
                    onRefresh: () {
                      _refreshBoard();
                    },
                    onPickDate: () {
                      _pickDate();
                    },
                    onPreviousDate: () {
                      _shiftDate(-1);
                    },
                    onNextDate: () {
                      _shiftDate(1);
                    },
                  ),
                  if (state.activityLabel != null) ...[
                    const SizedBox(height: 12),
                    const LinearProgressIndicator(),
                    const SizedBox(height: 8),
                    Text(
                      state.activityLabel!,
                      style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                            color: const Color(0xFF355B56),
                            fontWeight: FontWeight.w600,
                          ),
                    ),
                  ],
                  if (state.errorMessage != null) ...[
                    const SizedBox(height: 12),
                    _BannerCard(
                      tone: BannerTone.error,
                      message: state.errorMessage!,
                    ),
                  ],
                  if (state.noticeMessage != null) ...[
                    const SizedBox(height: 12),
                    _BannerCard(
                      tone: state.noticeTone == TaskBoardNoticeTone.info
                          ? BannerTone.info
                          : BannerTone.success,
                      message: state.noticeMessage!,
                    ),
                  ],
                  const SizedBox(height: 12),
                  ..._buildBoardSections(state),
                ],
              ),
            ),
          ),
        );
      },
    );
  }

  List<Widget> _buildBoardSections(TaskBoardViewState state) {
    final board = state.board;

    if (state.status == TaskBoardScreenStatus.loading && board == null) {
      return const <Widget>[_LoadingBoardCard()];
    }

    if (state.status == TaskBoardScreenStatus.error && board == null) {
      return <Widget>[
        _ErrorBoardCard(
          message: state.errorMessage ?? '加载失败，请稍后重试。',
          onRetry: () {
            _loadBoard(showLoadingState: true);
          },
        ),
      ];
    }

    if (board == null) {
      return const <Widget>[
        _EmptyBoard(
          title: '准备同步任务板',
          description: '填写 API、家庭成员和日期后，再开始同步今天的任务清单。',
        ),
      ];
    }

    final allSubjectGroups = board.groups;
    final subjectGroups = _controller.showCompletedHistory
        ? allSubjectGroups
        : allSubjectGroups
            .where((group) => group.status != 'completed')
            .toList();
    final pendingTaskCount =
        board.tasks.where((task) => !task.completed).length;
    final completedTaskCount =
        board.tasks.where((task) => task.completed).length;

    final sections = <Widget>[
      _SummaryPanel(
        summary: board.summary,
        date: board.date,
        busy: state.isBusy,
        lastSyncedAt: state.lastSyncedAt,
        activityLabel: state.activityLabel,
        onCompleteAll: () => _updateAllTasks(true),
        onResetAll: () => _updateAllTasks(false),
      ),
      const SizedBox(height: 12),
      _ChoiceHintPanel(
        pendingTaskCount: pendingTaskCount,
        completedTaskCount: completedTaskCount,
      ),
      const SizedBox(height: 12),
      _HistoryTogglePanel(
        completedTaskCount: completedTaskCount,
        showCompletedHistory: _controller.showCompletedHistory,
        onToggle: _controller.toggleCompletedHistory,
      ),
      const SizedBox(height: 12),
    ];

    if (state.status == TaskBoardScreenStatus.empty) {
      sections.add(
        const _EmptyBoard(
          title: '当前没有任务',
          description: '先让家长端完成任务解析和确认，再在这里同步今天的任务清单。',
        ),
      );
      return sections;
    }

    if (subjectGroups.isEmpty) {
      sections.add(
        _CompletedOnlyPanel(
          completedTaskCount: completedTaskCount,
          onShowHistory: () => _controller.toggleCompletedHistory(true),
        ),
      );
      return sections;
    }

    sections.addAll(
      subjectGroups.map((group) {
        final homeworkGroups = board.homeworkGroups
            .where((item) => item.subject == group.subject)
            .where(
              (item) =>
                  _controller.showCompletedHistory ||
                  item.status != 'completed',
            )
            .toList();
        final subjectTasks = board.tasks
            .where((task) => task.subject == group.subject)
            .where(
              (task) => _controller.showCompletedHistory || !task.completed,
            )
            .toList();

        return Padding(
          padding: const EdgeInsets.only(bottom: 12),
          child: _SubjectCard(
            group: group,
            homeworkGroups: homeworkGroups,
            tasks: subjectTasks,
            busy: state.isBusy,
            onToggleSubject: (completed) =>
                _updateSubjectGroup(group, completed),
            onToggleHomeworkGroup: _updateHomeworkGroup,
            expandedHomeworkGroupKeys: _controller.expandedHomeworkGroupKeys,
            onToggleHomeworkExpansion: _controller.toggleHomeworkGroupExpanded,
            onToggleTask: _updateSingleTask,
          ),
        );
      }),
    );
    return sections;
  }
}

class _ConfigPanel extends StatelessWidget {
  const _ConfigPanel({
    required this.apiBaseUrlController,
    required this.familyIdController,
    required this.userIdController,
    required this.dateController,
    required this.busy,
    required this.hasLoadedOnce,
    required this.lastSyncedAt,
    required this.onLoadBoard,
    required this.onRefresh,
    required this.onPickDate,
    required this.onPreviousDate,
    required this.onNextDate,
  });

  final TextEditingController apiBaseUrlController;
  final TextEditingController familyIdController;
  final TextEditingController userIdController;
  final TextEditingController dateController;
  final bool busy;
  final bool hasLoadedOnce;
  final DateTime? lastSyncedAt;
  final VoidCallback onLoadBoard;
  final VoidCallback onRefresh;
  final VoidCallback onPickDate;
  final VoidCallback onPreviousDate;
  final VoidCallback onNextDate;

  @override
  Widget build(BuildContext context) {
    final syncHint = lastSyncedAt == null
        ? '尚未成功同步'
        : '上次成功同步 ${_formatClock(lastSyncedAt!)}';

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              '同步配置',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.w700),
            ),
            const SizedBox(height: 6),
            const Text('Chrome 可直接访问 localhost，真机或 iPad 请改成局域网 API 地址。'),
            const SizedBox(height: 4),
            Text(
              syncHint,
              style: const TextStyle(
                color: Color(0xFF45635E),
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: apiBaseUrlController,
              keyboardType: TextInputType.url,
              decoration: const InputDecoration(
                labelText: 'API 地址',
                hintText: 'http://192.168.x.x:8080',
              ),
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: familyIdController,
                    keyboardType: TextInputType.number,
                    decoration: const InputDecoration(labelText: 'family_id'),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: TextField(
                    controller: userIdController,
                    keyboardType: TextInputType.number,
                    decoration: const InputDecoration(labelText: 'user_id'),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            Row(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                IconButton.filledTonal(
                  tooltip: '前一天',
                  onPressed: busy ? null : onPreviousDate,
                  icon: const Icon(Icons.chevron_left),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: TextField(
                    controller: dateController,
                    readOnly: true,
                    decoration: InputDecoration(
                      labelText: '任务日期',
                      hintText: '2026-03-06',
                      suffixIcon: IconButton(
                        tooltip: '选择日期',
                        onPressed: busy ? null : onPickDate,
                        icon: const Icon(Icons.calendar_month_outlined),
                      ),
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                IconButton.filledTonal(
                  tooltip: '下一天',
                  onPressed: busy ? null : onNextDate,
                  icon: const Icon(Icons.chevron_right),
                ),
              ],
            ),
            const SizedBox(height: 16),
            Wrap(
              spacing: 12,
              runSpacing: 12,
              children: [
                FilledButton.icon(
                  onPressed: busy ? null : onLoadBoard,
                  icon: const Icon(Icons.cloud_download_outlined),
                  label: Text(hasLoadedOnce ? '重新加载任务板' : '加载任务板'),
                ),
                OutlinedButton.icon(
                  onPressed: busy ? null : onRefresh,
                  icon: const Icon(Icons.refresh),
                  label: const Text('手动刷新'),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _SummaryPanel extends StatelessWidget {
  const _SummaryPanel({
    required this.summary,
    required this.date,
    required this.busy,
    required this.lastSyncedAt,
    required this.activityLabel,
    required this.onCompleteAll,
    required this.onResetAll,
  });

  final BoardSummary summary;
  final String date;
  final bool busy;
  final DateTime? lastSyncedAt;
  final String? activityLabel;
  final VoidCallback onCompleteAll;
  final VoidCallback onResetAll;

  @override
  Widget build(BuildContext context) {
    final syncLine = activityLabel ??
        (lastSyncedAt == null
            ? '尚未成功同步'
            : '最后同步 ${_formatClock(lastSyncedAt!)}');

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        '$date 任务板',
                        style: const TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text('整体状态: ${summary.statusLabel}'),
                      const SizedBox(height: 4),
                      Text(
                        syncLine,
                        style: TextStyle(
                          color: activityLabel == null
                              ? const Color(0xFF45635E)
                              : const Color(0xFF0F766E),
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ],
                  ),
                ),
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    OutlinedButton(
                      onPressed: busy ? null : onResetAll,
                      child: const Text('全部重置'),
                    ),
                    FilledButton(
                      onPressed: busy ? null : onCompleteAll,
                      child: const Text('全部完成'),
                    ),
                  ],
                ),
              ],
            ),
            const SizedBox(height: 16),
            Wrap(
              spacing: 12,
              runSpacing: 12,
              children: [
                _MetricChip(label: '总任务', value: '${summary.total}'),
                _MetricChip(label: '已完成', value: '${summary.completed}'),
                _MetricChip(label: '待完成', value: '${summary.pending}'),
                _MetricChip(label: '状态', value: summary.statusLabel),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _ChoiceHintPanel extends StatelessWidget {
  const _ChoiceHintPanel({
    required this.pendingTaskCount,
    required this.completedTaskCount,
  });

  final int pendingTaskCount;
  final int completedTaskCount;

  @override
  Widget build(BuildContext context) {
    return Card(
      color: const Color(0xFFE8F5F0),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              '自主安排任务',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.w800),
            ),
            const SizedBox(height: 6),
            if (pendingTaskCount == 0)
              Text('今天的任务已经全部完成，累计完成 $completedTaskCount 项。')
            else ...[
              Text('还剩 $pendingTaskCount 项待完成，你可以按自己的时间和状态自由安排。'),
              const SizedBox(height: 8),
              const Text('完成哪一项就直接勾选哪一项，系统只跟踪进度，不强制规定先后顺序。'),
            ],
          ],
        ),
      ),
    );
  }
}

class _HistoryTogglePanel extends StatelessWidget {
  const _HistoryTogglePanel({
    required this.completedTaskCount,
    required this.showCompletedHistory,
    required this.onToggle,
  });

  final int completedTaskCount;
  final bool showCompletedHistory;
  final ValueChanged<bool> onToggle;

  @override
  Widget build(BuildContext context) {
    if (completedTaskCount == 0) {
      return const SizedBox.shrink();
    }

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Text(
                    '已完成任务历史',
                    style: TextStyle(fontSize: 16, fontWeight: FontWeight.w700),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    showCompletedHistory
                        ? '当前正在显示 $completedTaskCount 条已完成任务。'
                        : '当前已隐藏 $completedTaskCount 条已完成任务。',
                  ),
                ],
              ),
            ),
            Switch(
              value: showCompletedHistory,
              onChanged: onToggle,
            ),
          ],
        ),
      ),
    );
  }
}

class _CompletedOnlyPanel extends StatelessWidget {
  const _CompletedOnlyPanel({
    required this.completedTaskCount,
    required this.onShowHistory,
  });

  final int completedTaskCount;
  final VoidCallback onShowHistory;

  @override
  Widget build(BuildContext context) {
    return Card(
      color: const Color(0xFFF0F7F4),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          children: [
            const Text(
              '未完成任务已经清空',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.w800),
            ),
            const SizedBox(height: 8),
            Text('今天已经完成 $completedTaskCount 条任务。'),
            const SizedBox(height: 12),
            FilledButton.tonal(
              onPressed: onShowHistory,
              child: const Text('查看已完成任务'),
            ),
          ],
        ),
      ),
    );
  }
}

class _SubjectCard extends StatelessWidget {
  const _SubjectCard({
    required this.group,
    required this.homeworkGroups,
    required this.tasks,
    required this.busy,
    required this.onToggleSubject,
    required this.onToggleHomeworkGroup,
    required this.expandedHomeworkGroupKeys,
    required this.onToggleHomeworkExpansion,
    required this.onToggleTask,
  });

  final TaskGroup group;
  final List<HomeworkGroup> homeworkGroups;
  final List<TaskItem> tasks;
  final bool busy;
  final ValueChanged<bool> onToggleSubject;
  final Future<void> Function(HomeworkGroup group, bool completed)
      onToggleHomeworkGroup;
  final Set<String> expandedHomeworkGroupKeys;
  final void Function(String subject, String groupTitle, bool expanded)
      onToggleHomeworkExpansion;
  final Future<void> Function(TaskItem task, bool completed) onToggleTask;

  @override
  Widget build(BuildContext context) {
    final subjectCompleted = group.status == 'completed';

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        group.subject,
                        style: const TextStyle(
                          fontSize: 20,
                          fontWeight: FontWeight.w800,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        '学科进度 ${group.completed}/${group.total}，还剩 ${group.pending} 项',
                      ),
                    ],
                  ),
                ),
                FilledButton.tonal(
                  onPressed:
                      busy ? null : () => onToggleSubject(!subjectCompleted),
                  child: Text(subjectCompleted ? '学科重置' : '学科完成'),
                ),
              ],
            ),
            const SizedBox(height: 14),
            ...homeworkGroups.map((homeworkGroup) {
              final homeworkTasks = tasks
                  .where((task) => task.groupTitle == homeworkGroup.groupTitle)
                  .toList();
              return Padding(
                padding: const EdgeInsets.only(bottom: 12),
                child: _HomeworkGroupCard(
                  group: homeworkGroup,
                  tasks: homeworkTasks,
                  isExpanded: expandedHomeworkGroupKeys.contains(
                    _groupKey(homeworkGroup),
                  ),
                  busy: busy,
                  onToggleGroup: (completed) =>
                      onToggleHomeworkGroup(homeworkGroup, completed),
                  onToggleExpand: (expanded) => onToggleHomeworkExpansion(
                    homeworkGroup.subject,
                    homeworkGroup.groupTitle,
                    expanded,
                  ),
                  onToggleTask: onToggleTask,
                ),
              );
            }),
          ],
        ),
      ),
    );
  }

  String _groupKey(HomeworkGroup group) {
    return '${group.subject}::${group.groupTitle}';
  }
}

class _HomeworkGroupCard extends StatelessWidget {
  const _HomeworkGroupCard({
    required this.group,
    required this.tasks,
    required this.isExpanded,
    required this.busy,
    required this.onToggleGroup,
    required this.onToggleExpand,
    required this.onToggleTask,
  });

  final HomeworkGroup group;
  final List<TaskItem> tasks;
  final bool isExpanded;
  final bool busy;
  final ValueChanged<bool> onToggleGroup;
  final ValueChanged<bool> onToggleExpand;
  final Future<void> Function(TaskItem task, bool completed) onToggleTask;

  @override
  Widget build(BuildContext context) {
    final groupCompleted = group.status == 'completed';

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: const Color(0xFFD6E5E0)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      group.groupTitle,
                      style: const TextStyle(
                        fontSize: 17,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      '分组进度 ${group.completed}/${group.total}，待完成 ${group.pending}',
                    ),
                  ],
                ),
              ),
              FilledButton.tonal(
                onPressed: busy ? null : () => onToggleGroup(!groupCompleted),
                child: Text(groupCompleted ? '分组重置' : '分组完成'),
              ),
              IconButton(
                onPressed: () => onToggleExpand(!isExpanded),
                icon: Icon(
                  isExpanded ? Icons.expand_less : Icons.expand_more,
                ),
                tooltip: isExpanded ? '收起任务' : '展开任务',
              ),
            ],
          ),
          if (isExpanded) ...[
            const SizedBox(height: 10),
            ...tasks.map(
              (task) => Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: CheckboxListTile(
                  value: task.completed,
                  onChanged: busy
                      ? null
                      : (value) {
                          if (value == null) {
                            return;
                          }
                          onToggleTask(task, value);
                        },
                  contentPadding: const EdgeInsets.symmetric(horizontal: 8),
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(12),
                  ),
                  title: Text(task.content),
                  subtitle: Text(
                    '任务 ${task.taskId} · ${task.completed ? "已完成" : "待完成"}',
                  ),
                  controlAffinity: ListTileControlAffinity.leading,
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _MetricChip extends StatelessWidget {
  const _MetricChip({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 108,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: const Color(0xFFE8F3EF),
        borderRadius: BorderRadius.circular(14),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label,
            style: const TextStyle(fontSize: 12, color: Color(0xFF45635E)),
          ),
          const SizedBox(height: 4),
          Text(
            value,
            style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w700),
          ),
        ],
      ),
    );
  }
}

class _LoadingBoardCard extends StatelessWidget {
  const _LoadingBoardCard();

  @override
  Widget build(BuildContext context) {
    return const Card(
      child: Padding(
        padding: EdgeInsets.all(24),
        child: Column(
          children: [
            CircularProgressIndicator(),
            SizedBox(height: 12),
            Text(
              '正在加载任务板',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.w700),
            ),
            SizedBox(height: 8),
            Text('正在向后端拉取当前日期的任务清单和同步状态。'),
          ],
        ),
      ),
    );
  }
}

class _ErrorBoardCard extends StatelessWidget {
  const _ErrorBoardCard({
    required this.message,
    required this.onRetry,
  });

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Card(
      color: const Color(0xFFFFF5F5),
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          children: [
            const Icon(Icons.error_outline, color: Color(0xFF9B1C1C), size: 32),
            const SizedBox(height: 12),
            const Text(
              '加载失败',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.w700),
            ),
            const SizedBox(height: 8),
            Text(
              message,
              textAlign: TextAlign.center,
              style: const TextStyle(color: Color(0xFF7F1D1D)),
            ),
            const SizedBox(height: 16),
            FilledButton.icon(
              onPressed: onRetry,
              icon: const Icon(Icons.refresh),
              label: const Text('重试加载'),
            ),
          ],
        ),
      ),
    );
  }
}

class _EmptyBoard extends StatelessWidget {
  const _EmptyBoard({
    required this.title,
    required this.description,
  });

  final String title;
  final String description;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          children: [
            Text(
              title,
              style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w700),
            ),
            const SizedBox(height: 8),
            Text(description, textAlign: TextAlign.center),
          ],
        ),
      ),
    );
  }
}

enum BannerTone { success, info, error }

class _BannerCard extends StatelessWidget {
  const _BannerCard({required this.tone, required this.message});

  final BannerTone tone;
  final String message;

  @override
  Widget build(BuildContext context) {
    final backgroundColor = tone == BannerTone.success
        ? const Color(0xFFE4F5EB)
        : tone == BannerTone.info
            ? const Color(0xFFE8F0FF)
            : const Color(0xFFFCE7E7);
    final foregroundColor = tone == BannerTone.success
        ? const Color(0xFF1F6B4D)
        : tone == BannerTone.info
            ? const Color(0xFF1D4ED8)
            : const Color(0xFF9B1C1C);

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: backgroundColor,
        borderRadius: BorderRadius.circular(14),
      ),
      child: Row(
        children: [
          Icon(
            tone == BannerTone.success
                ? Icons.check_circle_outline
                : tone == BannerTone.info
                    ? Icons.info_outline
                    : Icons.error_outline,
            color: foregroundColor,
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              message,
              style: TextStyle(
                color: foregroundColor,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

String _formatClock(DateTime value) {
  final hour = value.hour.toString().padLeft(2, '0');
  final minute = value.minute.toString().padLeft(2, '0');
  final second = value.second.toString().padLeft(2, '0');
  return '$hour:$minute:$second';
}
