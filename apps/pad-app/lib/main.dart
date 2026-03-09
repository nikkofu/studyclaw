import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';

const String defaultApiBaseUrl = String.fromEnvironment(
  'API_BASE_URL',
  defaultValue: 'http://localhost:8080',
);

void main() {
  runApp(const StudyClawPadApp());
}

class StudyClawPadApp extends StatelessWidget {
  const StudyClawPadApp({super.key, this.autoLoad = true});

  final bool autoLoad;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'StudyClaw Pad',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        useMaterial3: true,
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF0F766E)),
        scaffoldBackgroundColor: const Color(0xFFF3F7F5),
      ),
      home: PadTaskBoardPage(autoLoad: autoLoad),
    );
  }
}

class PadTaskBoardPage extends StatefulWidget {
  const PadTaskBoardPage({super.key, this.autoLoad = true});

  final bool autoLoad;

  @override
  State<PadTaskBoardPage> createState() => _PadTaskBoardPageState();
}

class _PadTaskBoardPageState extends State<PadTaskBoardPage> {
  final TextEditingController _apiBaseUrlController = TextEditingController(
    text: defaultApiBaseUrl,
  );
  final TextEditingController _familyIdController = TextEditingController(
    text: '306',
  );
  final TextEditingController _userIdController = TextEditingController(
    text: '1',
  );
  final TextEditingController _dateController = TextEditingController(
    text: '2026-03-06',
  );

  TaskBoard? _board;
  Set<String> _expandedHomeworkGroupKeys = <String>{};
  bool _showCompletedHistory = false;
  bool _loading = false;
  String? _errorMessage;
  String? _noticeMessage;

  @override
  void initState() {
    super.initState();
    if (widget.autoLoad) {
      _loadBoard();
    }
  }

  @override
  void dispose() {
    _apiBaseUrlController.dispose();
    _familyIdController.dispose();
    _userIdController.dispose();
    _dateController.dispose();
    super.dispose();
  }

  Future<void> _loadBoard() async {
    final requestContext = _buildRequestContext();
    if (requestContext == null) {
      return;
    }

    setState(() {
      _loading = true;
      _errorMessage = null;
      _noticeMessage = null;
    });

    try {
      final board =
          await TaskApiClient(baseUrl: requestContext.apiBaseUrl).fetchBoard(
        familyId: requestContext.familyId,
        userId: requestContext.userId,
        date: requestContext.date,
      );
      if (!mounted) {
        return;
      }
      _syncBoardState(board);
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _loading = false;
        _errorMessage = error.toString();
      });
    }
  }

  Future<void> _updateSingleTask(TaskItem task, bool completed) async {
    final requestContext = _buildRequestContext();
    if (requestContext == null) {
      return;
    }

    await _runBoardMutation(
      () => TaskApiClient(baseUrl: requestContext.apiBaseUrl).updateSingleTask(
        familyId: requestContext.familyId,
        userId: requestContext.userId,
        date: requestContext.date,
        taskId: task.taskId,
        completed: completed,
      ),
      completed ? '已同步单个任务完成状态' : '已恢复单个任务为待完成',
    );
  }

  Future<void> _updateSubjectGroup(TaskGroup group, bool completed) async {
    final requestContext = _buildRequestContext();
    if (requestContext == null) {
      return;
    }

    await _runBoardMutation(
      () => TaskApiClient(baseUrl: requestContext.apiBaseUrl).updateTaskGroup(
        familyId: requestContext.familyId,
        userId: requestContext.userId,
        date: requestContext.date,
        subject: group.subject,
        completed: completed,
      ),
      completed
          ? '已将 ${group.subject} 学科任务标记为完成'
          : '已将 ${group.subject} 学科任务恢复为待完成',
    );
  }

  Future<void> _updateHomeworkGroup(
    HomeworkGroup group,
    bool completed,
  ) async {
    final requestContext = _buildRequestContext();
    if (requestContext == null) {
      return;
    }

    await _runBoardMutation(
      () => TaskApiClient(baseUrl: requestContext.apiBaseUrl).updateTaskGroup(
        familyId: requestContext.familyId,
        userId: requestContext.userId,
        date: requestContext.date,
        subject: group.subject,
        groupTitle: group.groupTitle,
        completed: completed,
      ),
      completed
          ? '已将 ${group.groupTitle} 分组标记为完成'
          : '已将 ${group.groupTitle} 分组恢复为待完成',
    );
  }

  Future<void> _updateAllTasks(bool completed) async {
    final requestContext = _buildRequestContext();
    if (requestContext == null) {
      return;
    }

    await _runBoardMutation(
      () => TaskApiClient(baseUrl: requestContext.apiBaseUrl).updateAllTasks(
        familyId: requestContext.familyId,
        userId: requestContext.userId,
        date: requestContext.date,
        completed: completed,
      ),
      completed ? '已将全部任务同步为完成' : '已将全部任务恢复为待完成',
    );
  }

  Future<void> _runBoardMutation(
    Future<TaskBoard> Function() action,
    String successMessage,
  ) async {
    setState(() {
      _loading = true;
      _errorMessage = null;
      _noticeMessage = null;
    });

    try {
      final board = await action();
      if (!mounted) {
        return;
      }
      _syncBoardState(board, noticeMessage: successMessage);
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _loading = false;
        _errorMessage = error.toString();
      });
    }
  }

  void _syncBoardState(TaskBoard board, {String? noticeMessage}) {
    final expandedKeys = board.homeworkGroups
        .where((group) => group.status != 'completed')
        .map((group) => _homeworkGroupKey(group.subject, group.groupTitle))
        .toSet();

    setState(() {
      _board = board;
      _expandedHomeworkGroupKeys = expandedKeys;
      _loading = false;
      _noticeMessage = noticeMessage;
    });
  }

  void _toggleHomeworkGroupExpanded(
    String subject,
    String groupTitle,
    bool expanded,
  ) {
    final key = _homeworkGroupKey(subject, groupTitle);
    setState(() {
      if (expanded) {
        _expandedHomeworkGroupKeys.add(key);
      } else {
        _expandedHomeworkGroupKeys.remove(key);
      }
    });
  }

  String _homeworkGroupKey(String subject, String groupTitle) {
    return '$subject::$groupTitle';
  }

  void _toggleCompletedHistory(bool value) {
    setState(() {
      _showCompletedHistory = value;
    });
  }

  _RequestContext? _buildRequestContext() {
    final apiBaseUrl = _apiBaseUrlController.text.trim();
    final familyId = int.tryParse(_familyIdController.text.trim());
    final userId = int.tryParse(_userIdController.text.trim());
    final date = _dateController.text.trim();

    String? validationError;
    if (apiBaseUrl.isEmpty) {
      validationError = 'API 地址不能为空。';
    } else if (familyId == null || familyId <= 0) {
      validationError = 'family_id 需要是正整数。';
    } else if (userId == null || userId <= 0) {
      validationError = 'user_id 需要是正整数。';
    } else if (!RegExp(r'^\d{4}-\d{2}-\d{2}$').hasMatch(date)) {
      validationError = '日期需要是 YYYY-MM-DD。';
    }

    if (validationError != null) {
      setState(() {
        _errorMessage = validationError;
      });
      return null;
    }

    return _RequestContext(
      apiBaseUrl: apiBaseUrl,
      familyId: familyId!,
      userId: userId!,
      date: date,
    );
  }

  @override
  Widget build(BuildContext context) {
    final board = _board;
    final allSubjectGroups = board == null ? const <TaskGroup>[] : board.groups;
    final subjectGroups = _showCompletedHistory
        ? allSubjectGroups
        : allSubjectGroups
            .where((group) => group.status != 'completed')
            .toList();
    final pendingTaskCount =
        board == null ? 0 : board.tasks.where((task) => !task.completed).length;
    final completedTaskCount =
        board == null ? 0 : board.tasks.where((task) => task.completed).length;

    return Scaffold(
      appBar: AppBar(
        title: const Text('孩子任务同步台'),
        actions: [
          TextButton(
            onPressed: _loading ? null : _loadBoard,
            child: const Text('刷新任务'),
          ),
        ],
      ),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            _ConfigPanel(
              apiBaseUrlController: _apiBaseUrlController,
              familyIdController: _familyIdController,
              userIdController: _userIdController,
              dateController: _dateController,
              loading: _loading,
              onRefresh: _loadBoard,
            ),
            const SizedBox(height: 12),
            if (_loading) const LinearProgressIndicator(),
            if (_errorMessage != null) ...[
              const SizedBox(height: 12),
              _BannerCard(tone: BannerTone.error, message: _errorMessage!),
            ],
            if (_noticeMessage != null) ...[
              const SizedBox(height: 12),
              _BannerCard(tone: BannerTone.success, message: _noticeMessage!),
            ],
            const SizedBox(height: 12),
            if (board != null) ...[
              _SummaryPanel(
                summary: board.summary,
                date: board.date,
                loading: _loading,
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
                showCompletedHistory: _showCompletedHistory,
                onToggle: _toggleCompletedHistory,
              ),
              const SizedBox(height: 12),
              if (board.tasks.isEmpty)
                const _EmptyBoard()
              else if (subjectGroups.isEmpty)
                _CompletedOnlyPanel(
                  completedTaskCount: completedTaskCount,
                  onShowHistory: () => _toggleCompletedHistory(true),
                )
              else
                ...subjectGroups.map(
                  (group) {
                    final homeworkGroups = board.homeworkGroups
                        .where((item) => item.subject == group.subject)
                        .where(
                          (item) =>
                              _showCompletedHistory ||
                              item.status != 'completed',
                        )
                        .toList();
                    final subjectTasks = board.tasks
                        .where((task) => task.subject == group.subject)
                        .where(
                          (task) => _showCompletedHistory || !task.completed,
                        )
                        .toList();

                    return Padding(
                      padding: const EdgeInsets.only(bottom: 12),
                      child: _SubjectCard(
                        group: group,
                        homeworkGroups: homeworkGroups,
                        tasks: subjectTasks,
                        loading: _loading,
                        onToggleSubject: (completed) =>
                            _updateSubjectGroup(group, completed),
                        onToggleHomeworkGroup: _updateHomeworkGroup,
                        expandedHomeworkGroupKeys: _expandedHomeworkGroupKeys,
                        onToggleHomeworkExpansion: _toggleHomeworkGroupExpanded,
                        onToggleTask: _updateSingleTask,
                      ),
                    );
                  },
                ),
            ] else
              const _EmptyBoard(),
          ],
        ),
      ),
    );
  }
}

class _ConfigPanel extends StatelessWidget {
  const _ConfigPanel({
    required this.apiBaseUrlController,
    required this.familyIdController,
    required this.userIdController,
    required this.dateController,
    required this.loading,
    required this.onRefresh,
  });

  final TextEditingController apiBaseUrlController;
  final TextEditingController familyIdController;
  final TextEditingController userIdController;
  final TextEditingController dateController;
  final bool loading;
  final Future<void> Function() onRefresh;

  @override
  Widget build(BuildContext context) {
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
            const Text('孩子收到消息后，可以按日期拉取任务清单并同步完成情况。真机请改成局域网 API 地址。'),
            const SizedBox(height: 16),
            TextField(
              controller: apiBaseUrlController,
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
            TextField(
              controller: dateController,
              decoration: const InputDecoration(
                labelText: '任务日期',
                hintText: '2026-03-06',
              ),
            ),
            const SizedBox(height: 16),
            FilledButton(
              onPressed: loading ? null : onRefresh,
              child: const Text('加载任务板'),
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
    required this.loading,
    required this.onCompleteAll,
    required this.onResetAll,
  });

  final BoardSummary summary;
  final String date;
  final bool loading;
  final VoidCallback onCompleteAll;
  final VoidCallback onResetAll;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
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
                    ],
                  ),
                ),
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    OutlinedButton(
                      onPressed: loading ? null : onResetAll,
                      child: const Text('全部重置'),
                    ),
                    FilledButton(
                      onPressed: loading ? null : onCompleteAll,
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
    required this.loading,
    required this.onToggleSubject,
    required this.onToggleHomeworkGroup,
    required this.expandedHomeworkGroupKeys,
    required this.onToggleHomeworkExpansion,
    required this.onToggleTask,
  });

  final TaskGroup group;
  final List<HomeworkGroup> homeworkGroups;
  final List<TaskItem> tasks;
  final bool loading;
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
                      loading ? null : () => onToggleSubject(!subjectCompleted),
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
                  loading: loading,
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
    required this.loading,
    required this.onToggleGroup,
    required this.onToggleExpand,
    required this.onToggleTask,
  });

  final HomeworkGroup group;
  final List<TaskItem> tasks;
  final bool isExpanded;
  final bool loading;
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
                onPressed:
                    loading ? null : () => onToggleGroup(!groupCompleted),
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
                  onChanged: loading
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

class _EmptyBoard extends StatelessWidget {
  const _EmptyBoard();

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          children: const [
            Text(
              '当前没有任务',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.w700),
            ),
            SizedBox(height: 8),
            Text('先让家长端完成任务解析和确认，再在这里同步今天的任务清单。'),
          ],
        ),
      ),
    );
  }
}

enum BannerTone { success, error }

class _BannerCard extends StatelessWidget {
  const _BannerCard({required this.tone, required this.message});

  final BannerTone tone;
  final String message;

  @override
  Widget build(BuildContext context) {
    final Color backgroundColor = tone == BannerTone.success
        ? const Color(0xFFE4F5EB)
        : const Color(0xFFFCE7E7);
    final Color foregroundColor = tone == BannerTone.success
        ? const Color(0xFF1F6B4D)
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

class _RequestContext {
  const _RequestContext({
    required this.apiBaseUrl,
    required this.familyId,
    required this.userId,
    required this.date,
  });

  final String apiBaseUrl;
  final int familyId;
  final int userId;
  final String date;
}

class TaskApiClient {
  const TaskApiClient({required this.baseUrl});

  final String baseUrl;

  Future<TaskBoard> fetchBoard({
    required int familyId,
    required int userId,
    required String date,
  }) async {
    final payload = await _send(
      'GET',
      '/api/v1/tasks',
      query: {'family_id': '$familyId', 'user_id': '$userId', 'date': date},
    );
    return TaskBoard.fromJson(payload);
  }

  Future<TaskBoard> updateSingleTask({
    required int familyId,
    required int userId,
    required String date,
    required int taskId,
    required bool completed,
  }) async {
    final payload = await _send(
      'PATCH',
      '/api/v1/tasks/status/item',
      body: {
        'family_id': familyId,
        'assignee_id': userId,
        'task_id': taskId,
        'completed': completed,
        'assigned_date': date,
      },
    );
    return TaskBoard.fromJson(payload);
  }

  Future<TaskBoard> updateTaskGroup({
    required int familyId,
    required int userId,
    required String date,
    required String subject,
    String? groupTitle,
    required bool completed,
  }) async {
    final payload = await _send(
      'PATCH',
      '/api/v1/tasks/status/group',
      body: {
        'family_id': familyId,
        'assignee_id': userId,
        'subject': subject,
        if (groupTitle != null && groupTitle.isNotEmpty)
          'group_title': groupTitle,
        'completed': completed,
        'assigned_date': date,
      },
    );
    return TaskBoard.fromJson(payload);
  }

  Future<TaskBoard> updateAllTasks({
    required int familyId,
    required int userId,
    required String date,
    required bool completed,
  }) async {
    final payload = await _send(
      'PATCH',
      '/api/v1/tasks/status/all',
      body: {
        'family_id': familyId,
        'assignee_id': userId,
        'completed': completed,
        'assigned_date': date,
      },
    );
    return TaskBoard.fromJson(payload);
  }

  Future<Map<String, dynamic>> _send(
    String method,
    String path, {
    Map<String, String>? query,
    Map<String, dynamic>? body,
  }) async {
    final client = HttpClient();
    try {
      final uri = Uri.parse('$baseUrl$path').replace(queryParameters: query);
      final request = await client.openUrl(method, uri);
      request.headers.set(HttpHeaders.acceptHeader, 'application/json');
      if (body != null) {
        request.headers.contentType = ContentType.json;
        request.write(jsonEncode(body));
      }

      final response = await request.close();
      final payloadText = await utf8.decoder.bind(response).join();
      final payload = payloadText.isEmpty
          ? <String, dynamic>{}
          : jsonDecode(payloadText) as Map<String, dynamic>;

      if (response.statusCode < 200 || response.statusCode >= 300) {
        throw HttpException(
          payload['error']?.toString() ?? '请求失败，状态码 ${response.statusCode}',
          uri: uri,
        );
      }

      return payload;
    } finally {
      client.close(force: true);
    }
  }
}

class TaskBoard {
  const TaskBoard({
    required this.date,
    required this.tasks,
    required this.groups,
    required this.homeworkGroups,
    required this.summary,
    this.message,
  });

  factory TaskBoard.fromJson(Map<String, dynamic> json) {
    return TaskBoard(
      date: json['date']?.toString() ?? '',
      message: json['message']?.toString(),
      tasks: ((json['tasks'] as List<dynamic>? ?? const <dynamic>[]))
          .map((item) => TaskItem.fromJson(item as Map<String, dynamic>))
          .toList(),
      groups: ((json['groups'] as List<dynamic>? ?? const <dynamic>[]))
          .map((item) => TaskGroup.fromJson(item as Map<String, dynamic>))
          .toList(),
      homeworkGroups: ((json['homework_groups'] as List<dynamic>? ??
              const <dynamic>[]))
          .map((item) => HomeworkGroup.fromJson(item as Map<String, dynamic>))
          .toList(),
      summary: BoardSummary.fromJson(
        json['summary'] as Map<String, dynamic>? ?? const {},
      ),
    );
  }

  final String date;
  final String? message;
  final List<TaskItem> tasks;
  final List<TaskGroup> groups;
  final List<HomeworkGroup> homeworkGroups;
  final BoardSummary summary;
}

class TaskItem {
  const TaskItem({
    required this.taskId,
    required this.subject,
    required this.groupTitle,
    required this.content,
    required this.completed,
    required this.status,
  });

  factory TaskItem.fromJson(Map<String, dynamic> json) {
    return TaskItem(
      taskId: _readInt(json['task_id']),
      subject: json['subject']?.toString() ?? '未分类',
      groupTitle: _readString(json['group_title']).isNotEmpty
          ? _readString(json['group_title'])
          : _readString(json['content']),
      content: json['content']?.toString() ?? '',
      completed: json['completed'] == true,
      status: json['status']?.toString() ?? 'pending',
    );
  }

  final int taskId;
  final String subject;
  final String groupTitle;
  final String content;
  final bool completed;
  final String status;
}

class HomeworkGroup {
  const HomeworkGroup({
    required this.subject,
    required this.groupTitle,
    required this.total,
    required this.completed,
    required this.pending,
    required this.status,
  });

  factory HomeworkGroup.fromJson(Map<String, dynamic> json) {
    return HomeworkGroup(
      subject: json['subject']?.toString() ?? '未分类',
      groupTitle: json['group_title']?.toString() ?? '',
      total: _readInt(json['total']),
      completed: _readInt(json['completed']),
      pending: _readInt(json['pending']),
      status: json['status']?.toString() ?? 'pending',
    );
  }

  final String subject;
  final String groupTitle;
  final int total;
  final int completed;
  final int pending;
  final String status;
}

class TaskGroup {
  const TaskGroup({
    required this.subject,
    required this.total,
    required this.completed,
    required this.pending,
    required this.status,
  });

  factory TaskGroup.fromJson(Map<String, dynamic> json) {
    return TaskGroup(
      subject: json['subject']?.toString() ?? '未分类',
      total: _readInt(json['total']),
      completed: _readInt(json['completed']),
      pending: _readInt(json['pending']),
      status: json['status']?.toString() ?? 'pending',
    );
  }

  final String subject;
  final int total;
  final int completed;
  final int pending;
  final String status;
}

class BoardSummary {
  const BoardSummary({
    required this.total,
    required this.completed,
    required this.pending,
    required this.status,
  });

  factory BoardSummary.fromJson(Map<String, dynamic> json) {
    return BoardSummary(
      total: _readInt(json['total']),
      completed: _readInt(json['completed']),
      pending: _readInt(json['pending']),
      status: json['status']?.toString() ?? 'empty',
    );
  }

  final int total;
  final int completed;
  final int pending;
  final String status;

  String get statusLabel {
    switch (status) {
      case 'completed':
        return '全部完成';
      case 'partial':
        return '部分完成';
      case 'pending':
        return '待开始';
      default:
        return '空任务板';
    }
  }
}

int _readInt(Object? value) {
  if (value is int) {
    return value;
  }
  if (value is num) {
    return value.toInt();
  }
  return int.tryParse(value?.toString() ?? '') ?? 0;
}

String _readString(Object? value) {
  return value?.toString() ?? '';
}
