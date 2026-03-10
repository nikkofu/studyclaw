import 'package:flutter/material.dart';
import 'package:pad_app/task_board/controller.dart';
import 'package:pad_app/task_board/feedback.dart';
import 'package:pad_app/task_board/models.dart';
import 'package:pad_app/task_board/repository.dart';
import 'package:pad_app/task_board/weekly_stats.dart';
import 'package:pad_app/task_board/daily_stats.dart';
import 'package:pad_app/word_playback/controller.dart';
import 'package:pad_app/word_playback/models.dart';
import 'package:pad_app/word_playback/speaker.dart';

const String defaultApiBaseUrl = String.fromEnvironment(
  'API_BASE_URL',
  defaultValue: 'http://localhost:8080',
);

enum _PadHomeTab { tasks, words }

class PadTaskBoardPage extends StatefulWidget {
  const PadTaskBoardPage({
    super.key,
    this.autoLoad = true,
    this.initialDate,
    this.initialApiBaseUrl,
    this.initialFamilyId,
    this.initialUserId,
    this.repository = const RemoteTaskBoardRepository(),
    this.wordPlaybackController,
  });

  final bool autoLoad;
  final String? initialDate;
  final String? initialApiBaseUrl;
  final int? initialFamilyId;
  final int? initialUserId;
  final TaskBoardRepository repository;
  final WordPlaybackController? wordPlaybackController;

  @override
  State<PadTaskBoardPage> createState() => _PadTaskBoardPageState();
}

class _PadTaskBoardPageState extends State<PadTaskBoardPage> {
  late final TextEditingController _apiBaseUrlController;
  late final TextEditingController _familyIdController;
  late final TextEditingController _userIdController;
  late final TextEditingController _dateController;
  late final TextEditingController _wordListController;
  late final TaskBoardController _controller;
  late final WordPlaybackController _wordController;
  late final bool _ownsWordController;
  _PadHomeTab _selectedTab = _PadHomeTab.tasks;

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
    _ownsWordController = widget.wordPlaybackController == null;
    _wordController = widget.wordPlaybackController ??
        WordPlaybackController(
          speaker: createWordSpeaker(),
          repository: widget.repository,
        );
    _wordListController = TextEditingController(
      text: buildWordSampleText(_wordController.state.language),
    );
    if (!_wordController.state.hasWords) {
      _wordController.loadWordsFromText(_wordListController.text);
    }

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
    _wordListController.dispose();
    _controller.dispose();
    if (_ownsWordController) {
      _wordController.dispose();
    }
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

  void _setSelectedTab(_PadHomeTab tab) {
    if (_selectedTab == tab) {
      return;
    }
    setState(() {
      _selectedTab = tab;
    });
  }

  Future<void> _openSettingsSheet() async {
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      builder: (context) {
        final state = _controller.state;
        return Padding(
          padding: EdgeInsets.only(
            left: 16,
            right: 16,
            top: 8,
            bottom: MediaQuery.of(context).viewInsets.bottom + 16,
          ),
          child: SingleChildScrollView(
            child: _ConfigPanel(
              apiBaseUrlController: _apiBaseUrlController,
              familyIdController: _familyIdController,
              userIdController: _userIdController,
              dateController: _dateController,
              busy: state.isBusy,
              hasLoadedOnce: state.hasLoadedOnce,
              lastSyncedAt: state.lastSyncedAt,
              onLoadBoard: () {
                Navigator.of(context).pop();
                _loadBoard(showLoadingState: true);
              },
              onRefresh: () {
                Navigator.of(context).pop();
                _refreshBoard();
              },
              onPickDate: () async {
                Navigator.of(context).pop();
                await _pickDate();
              },
              onPreviousDate: () {
                Navigator.of(context).pop();
                _shiftDate(-1);
              },
              onNextDate: () {
                Navigator.of(context).pop();
                _shiftDate(1);
              },
            ),
          ),
        );
      },
    );
  }

  void _applyWordLanguage(WordPlaybackLanguage language) {
    _wordController.setLanguage(language);
    _wordListController.text = buildWordSampleText(language);
    _wordController.loadWordsFromText(_wordListController.text);
  }

  void _loadCurrentWordList() {
    _wordController.loadWordsFromText(_wordListController.text);
  }

  Future<void> _showDailyBriefSheet() async {
    final request = _buildRequest();
    if (request == null) {
      return;
    }

    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      builder: (context) {
        return FutureBuilder<DailyStats>(
          future: widget.repository.fetchDailyStats(request),
          builder: (context, snapshot) {
            if (snapshot.connectionState != ConnectionState.done) {
              return const _BottomSheetFrame(
                title: '今日简报',
                child: _InlineLoadingState(
                  title: '正在整理今日简报',
                  description: '正在向后端拉取今日任务执行与积分统计。',
                ),
              );
            }

            if (snapshot.hasError) {
              return _BottomSheetFrame(
                title: '今日简报',
                child: _InlineErrorState(
                  title: '今日简报暂时加载失败',
                  description: describePadApiFeedback(snapshot.error!).message,
                ),
              );
            }

            final dailyStats = snapshot.data!;
            final totals = dailyStats.totals;

            return _BottomSheetFrame(
              title: '今日简报',
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _ReportHero(
                    title: '今日进度 ${totals.completionRatePercent}%',
                    subtitle: dailyStats.encouragement,
                  ),
                  const SizedBox(height: 16),
                  Wrap(
                    spacing: 12,
                    runSpacing: 12,
                    children: [
                      _ExecutionMetricTile(
                        label: '今日积分',
                        value: totals.pointsDeltaLabel,
                        subtitle: '当前可用余额 ${totals.pointsBalance}',
                        icon: Icons.stars_outlined,
                      ),
                      _ExecutionMetricTile(
                        label: '今日完成',
                        value: '${totals.completedTasks}/${totals.totalTasks}',
                        subtitle: '还剩 ${totals.pendingTasks} 项任务',
                        icon: Icons.task_alt_outlined,
                      ),
                      _ExecutionMetricTile(
                        label: '听写练词',
                        value: '${totals.completedWordItems}/${totals.wordItems}',
                        subtitle: '共完成 ${totals.dictationSessions} 组听写',
                        icon: Icons.volume_up_outlined,
                      ),
                    ],
                  ),
                  const SizedBox(height: 16),
                  const Text(
                    '接下来做什么',
                    style: TextStyle(fontSize: 16, fontWeight: FontWeight.w700),
                  ),
                  const SizedBox(height: 8),
                  Text(totals.pendingTasks > 0 ? '还有任务没做完，建议优先完成列表里的待办项。' : '今天的任务已经全部完成了，可以开始复习或者休息啦。'),
                ],
              ),
            );
          },
        );
      },
    );
  }

  Future<void> _showWeeklyBriefSheet() async {
    final request = _buildRequest();
    if (request == null) {
      return;
    }

    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      builder: (context) {
        return FutureBuilder<WeeklyStats>(
          future: widget.repository.fetchWeeklyStats(request),
          builder: (context, snapshot) {
            if (snapshot.connectionState != ConnectionState.done) {
              return const _BottomSheetFrame(
                title: '本周鼓励',
                child: _InlineLoadingState(
                  title: '正在整理本周简报',
                  description: '正在向后端拉取最近 7 天的任务执行情况。',
                ),
              );
            }

            if (snapshot.hasError) {
              return _BottomSheetFrame(
                title: '本周鼓励',
                child: _InlineErrorState(
                  title: '本周简报暂时加载失败',
                  description:
                      describePadApiFeedback(snapshot.error!).message,
                ),
              );
            }

            final weeklyStats = snapshot.data!;
            if (!weeklyStats.hasData) {
              return const _BottomSheetFrame(
                title: '本周鼓励',
                child: _InlineEmptyState(
                  title: '本周还没有累计数据',
                  description: '先把今天的任务做起来，系统再为你整理本周鼓励。',
                ),
              );
            }

            return _BottomSheetFrame(
              title: '本周鼓励',
              child: _WeeklyBriefContent(stats: weeklyStats),
            );
          },
        );
      },
    );
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
      animation: Listenable.merge(<Listenable>[_controller, _wordController]),
      builder: (context, _) {
        final state = _controller.state;
        final board = state.board;

        return Scaffold(
          appBar: AppBar(
            title: const Text('StudyClaw 执行台'),
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
              IconButton(
                tooltip: '同步设置',
                onPressed: _openSettingsSheet,
                icon: const Icon(Icons.tune),
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
                  _TodayHeroCard(
                    date: _dateController.text.trim(),
                    summary: board?.summary,
                    lastSyncedAt: state.lastSyncedAt,
                    activityLabel: state.activityLabel,
                    onPreviousDate: state.isBusy ? null : () => _shiftDate(-1),
                    onNextDate: state.isBusy ? null : () => _shiftDate(1),
                    onPickDate: state.isBusy ? null : _pickDate,
                    onOpenSettings: _openSettingsSheet,
                    onCompleteAll:
                        board == null || state.isBusy ? null : () => _updateAllTasks(true),
                    onResetAll:
                        board == null || state.isBusy ? null : () => _updateAllTasks(false),
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
                  if (board != null && state.dailyStats != null) ...[
                    const SizedBox(height: 12),
                    _ExecutionOverviewCard(
                      stats: state.dailyStats!,
                      onOpenDailyBrief: _showDailyBriefSheet,
                      onOpenWeeklyBrief: _showWeeklyBriefSheet,
                    ),
                  ],
                  const SizedBox(height: 12),
                  _HomeModeSwitcher(
                    selectedTab: _selectedTab,
                    onChanged: _setSelectedTab,
                  ),
                  const SizedBox(height: 12),
                  if (_selectedTab == _PadHomeTab.tasks)
                    ..._buildBoardSections(state)
                  else
                    ...<Widget>[
                      _WordPlaybackPanel(
                        state: _wordController.state,
                        wordListController: _wordListController,
                        supportsPlayback: _wordController.supportsPlayback,
                        onLanguageChanged: _applyWordLanguage,
                        onLoadWords: _loadCurrentWordList,
                        onSyncBackend: () {
                          final request = _buildRequest();
                          if (request != null) {
                            _wordController.syncWordList(request);
                          }
                        },
                        onStartDictation: () {
                          final request = _buildRequest();
                          if (request != null) {
                            _wordController.startDictation(request);
                          }
                        },
                        onPlayCurrent: _wordController.playCurrent,
                        onReplayCurrent: () {
                          _wordController.replayCurrent(_apiBaseUrlController.text.trim());
                        },
                        onNextWord: () {
                          _wordController.nextWord(_apiBaseUrlController.text.trim());
                        },
                      ),
                    ],
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
          title: '准备开始今天的任务',
          description: '默认会加载当天任务；如果设备连不上后端，再到右上角调整同步设置。',
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



class _TodayHeroCard extends StatelessWidget {
  const _TodayHeroCard({
    required this.date,
    required this.summary,
    required this.lastSyncedAt,
    required this.activityLabel,
    required this.onPreviousDate,
    required this.onNextDate,
    required this.onPickDate,
    required this.onOpenSettings,
    required this.onCompleteAll,
    required this.onResetAll,
  });

  final String date;
  final BoardSummary? summary;
  final DateTime? lastSyncedAt;
  final String? activityLabel;
  final VoidCallback? onPreviousDate;
  final VoidCallback? onNextDate;
  final VoidCallback? onPickDate;
  final VoidCallback onOpenSettings;
  final VoidCallback? onCompleteAll;
  final VoidCallback? onResetAll;

  @override
  Widget build(BuildContext context) {
    final isToday = date == formatTaskBoardDate(DateTime.now());
    final title = isToday ? '今天任务板' : '$date 任务板';
    final boardSummary = summary;
    final subtitle = summary == null
        ? '默认会先加载当天任务，完成后直接勾选。'
        : boardSummary!.total == 0
            ? '今天暂时没有任务，先等家长端发布。'
            : '今天共有 ${boardSummary.total} 项任务，当前状态 ${boardSummary.statusLabel}。';
    final syncLine = activityLabel ??
        (lastSyncedAt == null
            ? '尚未成功同步'
            : '最后同步 ${_formatClock(lastSyncedAt!)}');

    return Container(
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          colors: <Color>[Color(0xFF0F766E), Color(0xFF15907F)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(24),
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
                      title,
                      key: const Key('today_hero_title'),
                      style: const TextStyle(
                        color: Colors.white,
                        fontSize: 24,
                        fontWeight: FontWeight.w800,
                      ),
                    ),
                    const SizedBox(height: 6),
                    Text(
                      subtitle,
                      style: const TextStyle(
                        color: Color(0xFFE2F7F1),
                        fontSize: 14,
                      ),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      syncLine,
                      style: const TextStyle(
                        color: Colors.white,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ],
                ),
              ),
              FilledButton.tonalIcon(
                onPressed: onOpenSettings,
                icon: const Icon(Icons.tune),
                label: const Text('设置'),
              ),
            ],
          ),
          const SizedBox(height: 16),
          Row(
            children: [
              IconButton.filledTonal(
                tooltip: '前一天',
                onPressed: onPreviousDate,
                icon: const Icon(Icons.chevron_left),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: onPickDate,
                  style: OutlinedButton.styleFrom(
                    foregroundColor: Colors.white,
                    side: const BorderSide(color: Color(0xFFB7E3DA)),
                  ),
                  icon: const Icon(Icons.calendar_month_outlined),
                  label: Text(date),
                ),
              ),
              const SizedBox(width: 8),
              IconButton.filledTonal(
                tooltip: '下一天',
                onPressed: onNextDate,
                icon: const Icon(Icons.chevron_right),
              ),
            ],
          ),
          const SizedBox(height: 12),
          Wrap(
            spacing: 12,
            runSpacing: 12,
            children: [
              OutlinedButton(
                key: const Key('today_hero_reset_all_button'),
                onPressed: onResetAll,
                style: OutlinedButton.styleFrom(
                  foregroundColor: Colors.white,
                  side: const BorderSide(color: Color(0xFFB7E3DA)),
                ),
                child: const Text('全部重置'),
              ),
              FilledButton(
                key: const Key('today_hero_complete_all_button'),
                onPressed: onCompleteAll,
                child: const Text('全部完成'),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _ExecutionOverviewCard extends StatelessWidget {
  const _ExecutionOverviewCard({
    required this.stats,
    required this.onOpenDailyBrief,
    required this.onOpenWeeklyBrief,
  });

  final DailyStats stats;
  final VoidCallback onOpenDailyBrief;
  final VoidCallback onOpenWeeklyBrief;

  @override
  Widget build(BuildContext context) {
    final totals = stats.totals;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              '今日执行概览',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.w800),
            ),
            const SizedBox(height: 6),
            Text(stats.encouragement),
            const SizedBox(height: 14),
            Wrap(
              spacing: 12,
              runSpacing: 12,
              children: [
                _ExecutionMetricTile(
                  label: '今日积分',
                  value: totals.pointsDeltaLabel,
                  subtitle: '当前余额 ${totals.pointsBalance}',
                  icon: Icons.stars_outlined,
                ),
                _ExecutionMetricTile(
                  label: '今日完成',
                  value: '${totals.completedTasks}/${totals.totalTasks}',
                  subtitle: '还剩 ${totals.pendingTasks} 项',
                  icon: Icons.task_alt_outlined,
                ),
                _ExecutionMetricTile(
                  label: '完成率',
                  value: '${totals.completionRatePercent}%',
                  subtitle: '练词完成 ${totals.completedWordItems}/${totals.wordItems}',
                  icon: Icons.show_chart,
                ),
              ],
            ),
            const SizedBox(height: 16),
            Container(
              padding: const EdgeInsets.all(14),
              decoration: BoxDecoration(
                color: const Color(0xFFF0F7F4),
                borderRadius: BorderRadius.circular(18),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Text(
                    '任务报告入口',
                    style: TextStyle(fontSize: 16, fontWeight: FontWeight.w700),
                  ),
                  const SizedBox(height: 6),
                  Text('今日进度 ${totals.completionRatePercent}%'),
                  const SizedBox(height: 6),
                  const Text(
                    '可查看今日详细积分明细或本周执行趋势。',
                    style: TextStyle(color: Color(0xFF45635E)),
                  ),
                  const SizedBox(height: 12),
                  Wrap(
                    spacing: 12,
                    runSpacing: 12,
                    children: [
                      FilledButton.tonalIcon(
                        onPressed: onOpenDailyBrief,
                        icon: const Icon(Icons.today_outlined),
                        label: const Text('今日简报'),
                      ),
                      OutlinedButton.icon(
                        onPressed: onOpenWeeklyBrief,
                        icon: const Icon(Icons.auto_graph_outlined),
                        label: const Text('本周鼓励'),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _ExecutionMetricTile extends StatelessWidget {
  const _ExecutionMetricTile({
    required this.label,
    required this.value,
    required this.subtitle,
    required this.icon,
  });

  final String label;
  final String value;
  final String subtitle;
  final IconData icon;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 172,
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: const Color(0xFFEAF4F1),
        borderRadius: BorderRadius.circular(18),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, color: const Color(0xFF0F766E)),
          const SizedBox(height: 12),
          Text(
            label,
            style: const TextStyle(color: Color(0xFF45635E)),
          ),
          const SizedBox(height: 4),
          Text(
            value,
            style: const TextStyle(fontSize: 22, fontWeight: FontWeight.w800),
          ),
          const SizedBox(height: 4),
          Text(
            subtitle,
            style: const TextStyle(fontSize: 12, color: Color(0xFF45635E)),
          ),
        ],
      ),
    );
  }
}

class _HomeModeSwitcher extends StatelessWidget {
  const _HomeModeSwitcher({
    required this.selectedTab,
    required this.onChanged,
  });

  final _PadHomeTab selectedTab;
  final ValueChanged<_PadHomeTab> onChanged;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(8),
        child: SegmentedButton<_PadHomeTab>(
          segments: const <ButtonSegment<_PadHomeTab>>[
            ButtonSegment<_PadHomeTab>(
              value: _PadHomeTab.tasks,
              icon: Icon(Icons.checklist_rtl),
              label: Text('今日任务'),
            ),
            ButtonSegment<_PadHomeTab>(
              value: _PadHomeTab.words,
              icon: Icon(Icons.volume_up_outlined),
              label: Text('单词播放'),
            ),
          ],
          selected: <_PadHomeTab>{selectedTab},
          onSelectionChanged: (selection) {
            onChanged(selection.first);
          },
        ),
      ),
    );
  }
}

class _WordPlaybackPanel extends StatelessWidget {
  const _WordPlaybackPanel({
    required this.state,
    required this.wordListController,
    required this.supportsPlayback,
    required this.onLanguageChanged,
    required this.onLoadWords,
    required this.onSyncBackend,
    required this.onStartDictation,
    required this.onPlayCurrent,
    required this.onReplayCurrent,
    required this.onNextWord,
  });

  final WordPlaybackState state;
  final TextEditingController wordListController;
  final bool supportsPlayback;
  final ValueChanged<WordPlaybackLanguage> onLanguageChanged;
  final VoidCallback onLoadWords;
  final VoidCallback onSyncBackend;
  final VoidCallback onStartDictation;
  final Future<void> Function() onPlayCurrent;
  final VoidCallback onReplayCurrent;
  final VoidCallback onNextWord;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              '单词播放模式',
              style: TextStyle(fontSize: 20, fontWeight: FontWeight.w800),
            ),
            const SizedBox(height: 6),
            Text(
              supportsPlayback
                  ? '当前词、重播、下一词都可以直接播放。'
                  : '当前设备暂不支持自动朗读，但仍可查看词卡、切换进度。',
            ),
            const SizedBox(height: 16),
            Row(
              children: [
                Expanded(
                  child: SegmentedButton<WordPlaybackLanguage>(
                    segments: const <ButtonSegment<WordPlaybackLanguage>>[
                      ButtonSegment<WordPlaybackLanguage>(
                        value: WordPlaybackLanguage.english,
                        label: Text('英语'),
                      ),
                      ButtonSegment<WordPlaybackLanguage>(
                        value: WordPlaybackLanguage.chinese,
                        label: Text('语文'),
                      ),
                    ],
                    selected: <WordPlaybackLanguage>{state.language},
                    onSelectionChanged: (selection) {
                      onLanguageChanged(selection.first);
                    },
                  ),
                ),
                const SizedBox(width: 12),
                FilledButton.icon(
                  onPressed: state.isBusy ? null : onSyncBackend,
                  icon: const Icon(Icons.cloud_sync_outlined),
                  label: const Text('同步词单'),
                ),
              ],
            ),
            const SizedBox(height: 16),
            TextField(
              controller: wordListController,
              minLines: 4,
              maxLines: 6,
              decoration: InputDecoration(
                labelText: '${state.language.label}词单',
                hintText: state.language.hintText,
                alignLabelWithHint: true,
              ),
            ),
            const SizedBox(height: 12),
            Wrap(
              spacing: 12,
              runSpacing: 12,
              children: [
                FilledButton.tonalIcon(
                  onPressed: onLoadWords,
                  icon: const Icon(Icons.playlist_add_check_circle_outlined),
                  label: const Text('本地更新'),
                ),
                OutlinedButton.icon(
                  onPressed: () => onLanguageChanged(state.language),
                  icon: const Icon(Icons.auto_fix_high_outlined),
                  label: const Text('载入示例'),
                ),
              ],
            ),
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
                tone: BannerTone.info,
                message: state.noticeMessage!,
              ),
            ],
            const SizedBox(height: 16),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(20),
              decoration: BoxDecoration(
                color: const Color(0xFFF4F8F7),
                borderRadius: BorderRadius.circular(22),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      const Text(
                        '当前词',
                        style: TextStyle(
                          fontSize: 15,
                          fontWeight: FontWeight.w700,
                          color: Color(0xFF45635E),
                        ),
                      ),
                      if (state.session == null)
                        FilledButton.icon(
                          onPressed: state.isBusy || !state.hasWords ? null : onStartDictation,
                          style: FilledButton.styleFrom(
                            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 0),
                          ),
                          icon: const Icon(Icons.play_circle_outline, size: 18),
                          label: const Text('开启听写会话'),
                        )
                      else
                        const Chip(
                          label: Text('听写进行中'),
                          backgroundColor: Color(0xFFE2F7F1),
                          labelStyle: TextStyle(fontSize: 12, color: Color(0xFF0F766E)),
                        ),
                    ],
                  ),
                  const SizedBox(height: 10),
                  Text(
                    state.hasWords ? state.currentWord : '请先同步或输入词单',
                    style: const TextStyle(
                      fontSize: 32,
                      fontWeight: FontWeight.w800,
                    ),
                  ),
                  const SizedBox(height: 16),
                  LinearProgressIndicator(
                    value: state.progress,
                    borderRadius: BorderRadius.circular(999),
                    minHeight: 10,
                  ),
                  const SizedBox(height: 8),
                  Text(
                    '播放进度 ${state.currentDisplayIndex}/${state.totalWords}',
                    style: const TextStyle(color: Color(0xFF45635E)),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 16),
            Wrap(
              spacing: 12,
              runSpacing: 12,
              children: [
                FilledButton.icon(
                  onPressed: state.hasWords && !state.isBusy ? onPlayCurrent : null,
                  icon: const Icon(Icons.volume_up_outlined),
                  label: Text(state.isSpeaking ? '播放中' : '播放当前'),
                ),
                OutlinedButton.icon(
                  onPressed: state.hasWords && !state.isBusy ? onReplayCurrent : null,
                  icon: const Icon(Icons.replay),
                  label: const Text('重播'),
                ),
                FilledButton.tonalIcon(
                  onPressed: state.hasWords && !state.isBusy ? onNextWord : null,
                  icon: const Icon(Icons.skip_next),
                  label: const Text('下一词'),
                ),
              ],
            ),
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
                        key: const Key('summary_panel_title'),
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
                      key: const Key('summary_panel_reset_all_button'),
                      onPressed: busy ? null : onResetAll,
                      child: const Text('全部重置'),
                    ),
                    FilledButton(
                      key: const Key('summary_panel_complete_all_button'),
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

class _BottomSheetFrame extends StatelessWidget {
  const _BottomSheetFrame({
    required this.title,
    required this.child,
  });

  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 8, 20, 24),
        child: SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                title,
                style: const TextStyle(fontSize: 22, fontWeight: FontWeight.w800),
              ),
              const SizedBox(height: 16),
              child,
            ],
          ),
        ),
      ),
    );
  }
}

class _ReportHero extends StatelessWidget {
  const _ReportHero({
    required this.title,
    required this.subtitle,
  });

  final String title;
  final String subtitle;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        color: const Color(0xFFEAF4F1),
        borderRadius: BorderRadius.circular(20),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            title,
            style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w800),
          ),
          const SizedBox(height: 8),
          Text(
            subtitle,
            style: const TextStyle(color: Color(0xFF355B56)),
          ),
        ],
      ),
    );
  }
}

class _InlineLoadingState extends StatelessWidget {
  const _InlineLoadingState({
    required this.title,
    required this.description,
  });

  final String title;
  final String description;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        const Center(child: CircularProgressIndicator()),
        const SizedBox(height: 16),
        Text(
          title,
          style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w700),
        ),
        const SizedBox(height: 8),
        Text(description, textAlign: TextAlign.center),
      ],
    );
  }
}

class _InlineErrorState extends StatelessWidget {
  const _InlineErrorState({
    required this.title,
    required this.description,
  });

  final String title;
  final String description;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        const Icon(Icons.error_outline, size: 32, color: Color(0xFF9B1C1C)),
        const SizedBox(height: 12),
        Text(
          title,
          style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w700),
        ),
        const SizedBox(height: 8),
        Text(description, textAlign: TextAlign.center),
      ],
    );
  }
}

class _InlineEmptyState extends StatelessWidget {
  const _InlineEmptyState({
    required this.title,
    required this.description,
  });

  final String title;
  final String description;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Text(
          title,
          style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w700),
        ),
        const SizedBox(height: 8),
        Text(description, textAlign: TextAlign.center),
      ],
    );
  }
}

class _WeeklyBriefContent extends StatelessWidget {
  const _WeeklyBriefContent({required this.stats});

  final WeeklyStats stats;

  @override
  Widget build(BuildContext context) {
    final insight = stats.insight;
    final summary = insight?.summary.trim().isNotEmpty == true
        ? insight!.summary
        : '本周一共完成了 ${stats.completedTasks}/${stats.totalTasks} 项任务，继续保持这个节奏。';

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _ReportHero(
          title: '本周完成率 ${stats.completionRatePercent}%',
          subtitle: summary,
        ),
        const SizedBox(height: 16),
        Wrap(
          spacing: 12,
          runSpacing: 12,
          children: [
            _ExecutionMetricTile(
              label: '本周任务',
              value: '${stats.totalTasks}',
              subtitle: '${stats.days.length} 个活跃日',
              icon: Icons.calendar_view_week_outlined,
            ),
            _ExecutionMetricTile(
              label: '已完成',
              value: '${stats.completedTasks}',
              subtitle: '待完成 ${stats.pendingTasks}',
              icon: Icons.done_all_outlined,
            ),
          ],
        ),
        if (insight != null) ...[
          const SizedBox(height: 20),
          const Text(
            '本周做得好的地方',
            style: TextStyle(fontSize: 16, fontWeight: FontWeight.w700),
          ),
          const SizedBox(height: 8),
          ...insight.strengths.map(
            (item) => Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: _SheetBullet(
                icon: Icons.check_circle_outline,
                text: item,
              ),
            ),
          ),
          const SizedBox(height: 12),
          const Text(
            '下周继续加强',
            style: TextStyle(fontSize: 16, fontWeight: FontWeight.w700),
          ),
          const SizedBox(height: 8),
          ...insight.areasForImprovement.map(
            (item) => Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: _SheetBullet(
                icon: Icons.flag_outlined,
                text: item,
              ),
            ),
          ),
          const SizedBox(height: 12),
          _ReportHero(
            title: '成长提醒',
            subtitle: insight.psychologicalInsight,
          ),
        ],
      ],
    );
  }
}

class _SheetBullet extends StatelessWidget {
  const _SheetBullet({
    required this.icon,
    required this.text,
  });

  final IconData icon;
  final String text;

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Icon(icon, color: const Color(0xFF0F766E), size: 20),
        const SizedBox(width: 10),
        Expanded(child: Text(text)),
      ],
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
