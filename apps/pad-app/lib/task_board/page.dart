import 'dart:convert';
import 'dart:async';
import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:pad_app/ui_kit/kid_theme.dart';
import 'package:pad_app/ui_kit/kid_card.dart';
import 'package:pad_app/ui_kit/kid_components.dart';
import 'package:pad_app/task_board/controller.dart';
import 'package:pad_app/task_board/models.dart';
import 'package:pad_app/task_board/repository.dart';
import 'package:pad_app/word_playback/controller.dart';
import 'package:pad_app/word_playback/models.dart';
import 'package:pad_app/word_playback/speaker.dart';

const String defaultApiBaseUrl = String.fromEnvironment('API_BASE_URL',
    defaultValue: 'http://localhost:8080');

enum _PadHomeTab { tasks, words }

enum _JourneyStepVisualState { complete, active, pending, failed }

class _JourneyStepData {
  const _JourneyStepData({
    required this.title,
    required this.caption,
    required this.state,
    this.timeLabel,
  });

  final String title;
  final String caption;
  final _JourneyStepVisualState state;
  final String? timeLabel;
}

String _formatJourneyTime(String? raw) {
  final value = raw?.trim() ?? '';
  if (value.isEmpty) return '等待中';

  final parsed = DateTime.tryParse(value);
  if (parsed == null) return value;
  final local = parsed.toLocal();
  final hour = local.hour.toString().padLeft(2, '0');
  final minute = local.minute.toString().padLeft(2, '0');
  return '$hour:$minute';
}

String _formatSubmissionClock(DateTime submittedAt) {
  final local = submittedAt.toLocal();
  final hour = local.hour.toString().padLeft(2, '0');
  final minute = local.minute.toString().padLeft(2, '0');
  return '提交 $hour:$minute';
}

String _formatSubmissionBytes(int byteCount) {
  if (byteCount < 1024) {
    return '大小 $byteCount B';
  }

  final kilobytes = byteCount / 1024;
  if (kilobytes < 1024) {
    final value = kilobytes >= 10
        ? kilobytes.toStringAsFixed(0)
        : kilobytes.toStringAsFixed(1);
    return '大小 $value KB';
  }

  final megabytes = kilobytes / 1024;
  return '大小 ${megabytes.toStringAsFixed(1)} MB';
}

String _shortTraceToken(String? raw) {
  final value = raw?.trim() ?? '';
  if (value.isEmpty) return '';
  return value.length <= 8 ? value : value.substring(0, 8);
}

List<_JourneyStepData> _buildJourneySteps(DictationSession session) {
  final stageKey = resolveDictationWorkerStage(session);

  _JourneyStepVisualState receiveState;
  switch (stageKey) {
    case 'queued':
      receiveState = _JourneyStepVisualState.active;
      break;
    case 'mark_processing_failed':
      receiveState = _JourneyStepVisualState.failed;
      break;
    case 'idle':
      receiveState = _JourneyStepVisualState.pending;
      break;
    default:
      receiveState = _JourneyStepVisualState.complete;
  }

  _JourneyStepVisualState compareState;
  switch (stageKey) {
    case 'processing':
    case 'loading_word_list':
    case 'llm_grading':
      compareState = _JourneyStepVisualState.active;
      break;
    case 'load_word_list_failed':
    case 'llm_grading_failed':
      compareState = _JourneyStepVisualState.failed;
      break;
    case 'completed':
    case 'persist_result_failed':
      compareState = _JourneyStepVisualState.complete;
      break;
    default:
      compareState = receiveState == _JourneyStepVisualState.complete
          ? _JourneyStepVisualState.pending
          : _JourneyStepVisualState.pending;
  }

  _JourneyStepVisualState resultState;
  switch (stageKey) {
    case 'completed':
      resultState = _JourneyStepVisualState.complete;
      break;
    case 'persist_result_failed':
    case 'failed':
      resultState = _JourneyStepVisualState.failed;
      break;
    default:
      resultState = _JourneyStepVisualState.pending;
  }

  return <_JourneyStepData>[
    _JourneyStepData(
      title: '拍照完成',
      caption: '照片已经准备好，正在送去批改。',
      state: _JourneyStepVisualState.complete,
      timeLabel: _formatJourneyTime(
          session.gradingRequestedAt ?? session.updatedAt ?? session.startedAt),
    ),
    _JourneyStepData(
      title: '云端接收',
      caption: '后台接到这次交卷请求。',
      state: receiveState,
      timeLabel: _formatJourneyTime(session.gradingRequestedAt),
    ),
    _JourneyStepData(
      title: 'AI 比对',
      caption: 'AI 看照片、核对正确答案。',
      state: compareState,
      timeLabel: compareState == _JourneyStepVisualState.complete ||
              compareState == _JourneyStepVisualState.failed ||
              compareState == _JourneyStepVisualState.active
          ? _formatJourneyTime(session.gradingCompletedAt ??
              session.updatedAt ??
              session.gradingRequestedAt)
          : null,
    ),
    _JourneyStepData(
      title: '结果回到平板',
      caption: '最终结果同步回当前设备。',
      state: resultState,
      timeLabel: resultState == _JourneyStepVisualState.complete ||
              resultState == _JourneyStepVisualState.failed
          ? _formatJourneyTime(session.gradingCompletedAt ?? session.updatedAt)
          : null,
    ),
  ];
}

List<String> _buildTracePills(DictationSession session) {
  final pills = <String>[
    if (session.sessionId.trim().isNotEmpty) '会话 ${session.sessionId}',
    if (session.gradingResult?.gradingId.trim().isNotEmpty == true)
      '批改 ${session.gradingResult!.gradingId}',
    if (_shortTraceToken(session.debugContext?.photoSha1).isNotEmpty)
      '图片 ${_shortTraceToken(session.debugContext?.photoSha1)}',
  ];
  return pills;
}

class PadTaskBoardPage extends StatefulWidget {
  const PadTaskBoardPage(
      {super.key,
      this.autoLoad = true,
      this.initialDate,
      this.initialApiBaseUrl,
      this.initialFamilyId,
      this.initialUserId,
      this.repository = const RemoteTaskBoardRepository(),
      this.wordPlaybackController});
  final bool autoLoad;
  final String? initialDate, initialApiBaseUrl;
  final int? initialFamilyId, initialUserId;
  final TaskBoardRepository repository;
  final WordPlaybackController? wordPlaybackController;
  @override
  State<PadTaskBoardPage> createState() => _PadTaskBoardPageState();
}

class _PadTaskBoardPageState extends State<PadTaskBoardPage>
    with SingleTickerProviderStateMixin {
  late final AnimationController _celebrationController;
  int _selectedSubjectIndex = 0;
  late final TextEditingController _apiBaseUrlController,
      _familyIdController,
      _userIdController,
      _dateController,
      _wordListController;
  late final TaskBoardController _controller;
  late final WordPlaybackController _wordController;
  late final bool _ownsWordController;
  _PadHomeTab _selectedTab = _PadHomeTab.tasks;

  @override
  void initState() {
    super.initState();
    _celebrationController =
        AnimationController(vsync: this, duration: const Duration(seconds: 1));
    _apiBaseUrlController = TextEditingController(
        text: widget.initialApiBaseUrl ?? defaultApiBaseUrl);
    _familyIdController =
        TextEditingController(text: '${widget.initialFamilyId ?? 306}');
    _userIdController =
        TextEditingController(text: '${widget.initialUserId ?? 1}');
    _dateController = TextEditingController(
        text: widget.initialDate ?? formatTaskBoardDate(DateTime.now()));
    _wordListController = TextEditingController();
    _controller = TaskBoardController(repository: widget.repository);
    if (widget.wordPlaybackController != null) {
      _wordController = widget.wordPlaybackController!;
      _ownsWordController = false;
    } else {
      _wordController = WordPlaybackController(
          speaker: createWordSpeaker(), repository: widget.repository);
      _ownsWordController = true;
    }
    if (widget.autoLoad) {
      scheduleMicrotask(() => _loadBoard(showLoadingState: true));
    }
  }

  @override
  void dispose() {
    _celebrationController.dispose();
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

  TaskBoardRequest? _buildRequest() {
    final req = TaskBoardRequest(
        apiBaseUrl: _apiBaseUrlController.text.trim(),
        familyId: int.tryParse(_familyIdController.text.trim()) ?? 0,
        userId: int.tryParse(_userIdController.text.trim()) ?? 0,
        date: _dateController.text.trim());
    return req.validate() == null ? req : null;
  }

  void _loadBoard({bool showLoadingState = false}) {
    final r = _buildRequest();
    if (r != null) {
      _controller.loadBoard(r, showLoadingState: showLoadingState);
    }
  }

  Future<void> _refreshBoard() async {
    final r = _buildRequest();
    if (r != null) {
      await _controller.refresh(r);
    }
  }

  void _setSelectedTab(_PadHomeTab tab) => setState(() => _selectedTab = tab);

  Future<void> _updateSingleTask(TaskItem task, bool completed) async {
    final r = _buildRequest();
    if (r == null) {
      return;
    }
    await _controller.updateSingleTask(r, task, completed);
    if (completed &&
        _controller.state.board?.tasks.every((t) => t.completed) == true) {
      _celebrationController.forward(from: 0);
    }
  }

  Future<void> _updateAllTasks(bool completed) async {
    final r = _buildRequest();
    if (r == null) {
      return;
    }
    await _controller.updateAllTasks(r, completed: completed);
    if (completed) {
      _celebrationController.forward(from: 0);
    }
  }

  Future<void> _takeAndSubmitPhoto() async {
    final picker = ImagePicker();
    final photo = await picker.pickImage(
        source: ImageSource.camera, maxWidth: 1024, imageQuality: 85);
    if (photo == null) return;
    final r = _buildRequest();
    if (r == null) return;

    // Auto-start session if not active
    if (_wordController.state.session == null) {
      await _wordController.startDictation(r);
    }

    final submittedAt = DateTime.now();
    final bytes = await photo.readAsBytes();
    final base64Image = base64Encode(bytes);
    await _wordController.submitPhotoForGrading(
      r.apiBaseUrl,
      base64Image,
      previewBytes: bytes,
      submittedAt: submittedAt,
    );
  }

  void _openSettingsSheet() {
    showModalBottomSheet<void>(
        context: context,
        isScrollControlled: true,
        builder: (ctx) => KidBottomSheetFrame(
            title: '同步配置',
            child: Column(children: [
              TextField(
                  controller: _apiBaseUrlController,
                  decoration: const InputDecoration(labelText: 'API 地址')),
              const SizedBox(height: 16),
              Row(children: [
                Expanded(
                    child: TextField(
                        controller: _familyIdController,
                        decoration: const InputDecoration(labelText: '家庭 ID'))),
                const SizedBox(width: 16),
                Expanded(
                    child: TextField(
                        controller: _userIdController,
                        decoration: const InputDecoration(labelText: '孩子 ID')))
              ]),
              const SizedBox(height: 24),
              KidActionBtn(
                  label: '立即同步',
                  color: KidColors.color1,
                  onTap: () {
                    Navigator.pop(ctx);
                    _loadBoard(showLoadingState: true);
                  }),
            ])));
  }

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
        listenable: _controller,
        builder: (context, _) {
          final state = _controller.state;
          return Scaffold(
            appBar: AppBar(title: const Text('StudyClaw Pad'), actions: [
              IconButton(
                  onPressed: _refreshBoard,
                  icon: const Icon(Icons.refresh_rounded)),
              IconButton(
                  onPressed: _openSettingsSheet,
                  icon: const Icon(Icons.tune_rounded)),
              const SizedBox(width: 8)
            ]),
            body: SafeArea(
                child: Stack(children: [
              RefreshIndicator(
                  onRefresh: _refreshBoard,
                  child: ListView(padding: const EdgeInsets.all(24), children: [
                    _TodayHeroCard(
                        date: _dateController.text.trim(),
                        onPreviousDate: () => _shiftDate(-1),
                        onNextDate: () => _shiftDate(1),
                        onCompleteAll: () => _updateAllTasks(true),
                        onResetAll: () => _updateAllTasks(false)),
                    if (state.errorMessage != null) ...[
                      const SizedBox(height: 16),
                      _BannerCard(message: state.errorMessage!)
                    ],
                    const SizedBox(height: 24),
                    _HomeModeSwitcher(
                        selectedTab: _selectedTab, onChanged: _setSelectedTab),
                    const SizedBox(height: 24),
                    if (_selectedTab == _PadHomeTab.tasks)
                      ..._buildBoardSections(state)
                    else
                      ListenableBuilder(
                        listenable: _wordController,
                        builder: (context, _) {
                          final wState = _wordController.state;
                          return _WordPlaybackPanel(
                            state: wState,
                            onLanguageChanged: _wordController.setLanguage,
                            onModeChanged: _wordController.setMode,
                            onPeekChanged: _wordController.setPeeking,
                            onSyncBackend: () {
                              final r = _buildRequest();
                              if (r != null) _wordController.syncWordList(r);
                            },
                            onPlayCurrent: () async {
                              if (_wordController.state.session == null) {
                                final r = _buildRequest();
                                if (r != null) {
                                  await _wordController.startDictation(r);
                                  return;
                                }
                              }
                              await _wordController.playCurrent();
                            },
                            onReplayCurrent: () => _wordController
                                .replayCurrent(_apiBaseUrlController.text),
                            onNextWord: () => _wordController
                                .nextWord(_apiBaseUrlController.text),
                            onPreviousWord: () => _wordController
                                .previousWord(_apiBaseUrlController.text),
                            onSubmitPhoto: _takeAndSubmitPhoto,
                          );
                        },
                      ),
                  ])),
              Align(
                  alignment: Alignment.center,
                  child: IgnorePointer(
                      child: AnimatedBuilder(
                          animation: _celebrationController,
                          builder: (context, _) {
                            if (_celebrationController.value == 0) {
                              return const SizedBox.shrink();
                            }
                            return Opacity(
                                opacity: (1.0 - _celebrationController.value)
                                    .clamp(0.0, 1.0),
                                child: Transform.scale(
                                    scale: 1.0 +
                                        _celebrationController.value * 2.0,
                                    child: const Icon(Icons.stars_rounded,
                                        color: KidColors.color4, size: 160)));
                          }))),
            ])),
          );
        });
  }

  void _shiftDate(int d) {
    _dateController.text = formatTaskBoardDate(
        (parseTaskBoardDate(_dateController.text.trim()) ?? DateTime.now())
            .add(Duration(days: d)));
    _loadBoard(showLoadingState: true);
  }

  List<Widget> _buildBoardSections(TaskBoardViewState state) {
    final board = state.board;
    if (board == null) {
      return [const KidInlineLoading(title: '同步中', description: '正在准备挑战舞台...')];
    }
    final groups = board.groups;
    if (groups.isEmpty) {
      return [const _EmptyBoard(title: '暂无任务', description: '表现真棒！')];
    }
    if (_selectedSubjectIndex >= groups.length) {
      _selectedSubjectIndex = 0;
    }
    final focused = groups[_selectedSubjectIndex];
    return [
      _SubjectNavigator(
          groups: groups,
          selectedIndex: _selectedSubjectIndex,
          onSelected: (i) => setState(() => _selectedSubjectIndex = i)),
      const SizedBox(height: 24),
      _FocusedSubjectStage(
          group: focused,
          homeworkGroups: board.homeworkGroups
              .where((h) => h.subject == focused.subject)
              .toList(),
          tasks:
              board.tasks.where((t) => t.subject == focused.subject).toList(),
          busy: state.isBusy,
          onToggleTask: _updateSingleTask),
    ];
  }
}

class _TodayHeroCard extends StatelessWidget {
  const _TodayHeroCard(
      {required this.date,
      required this.onPreviousDate,
      required this.onNextDate,
      required this.onCompleteAll,
      required this.onResetAll});
  final String date;
  final VoidCallback? onPreviousDate, onNextDate, onCompleteAll, onResetAll;
  @override
  Widget build(BuildContext context) => KidCard(
      color: KidColors.color1,
      hasBorder: false,
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          Expanded(
              child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                Text(date,
                    style: const TextStyle(
                        color: KidColors.white,
                        fontSize: 16,
                        fontWeight: FontWeight.w700)),
                const Text('挑战舞台',
                    style: TextStyle(
                        color: KidColors.white,
                        fontSize: 32,
                        fontWeight: FontWeight.w900))
              ])),
          const Icon(Icons.rocket_launch_rounded,
              color: KidColors.white, size: 40)
        ]),
        const SizedBox(height: 24),
        Row(children: [
          IconButton(
              onPressed: onPreviousDate,
              icon:
                  const Icon(Icons.chevron_left_rounded, color: Colors.white)),
          const Spacer(),
          const Text('前进！向终点冲刺',
              style:
                  TextStyle(color: Colors.white, fontWeight: FontWeight.w800)),
          const Spacer(),
          IconButton(
              onPressed: onNextDate,
              icon:
                  const Icon(Icons.chevron_right_rounded, color: Colors.white))
        ]),
        const SizedBox(height: 24),
        Row(children: [
          Expanded(
              child: KidSmallBtn(
                  label: '全部重置', color: KidColors.color5, onTap: onResetAll)),
          const SizedBox(width: 12),
          Expanded(
              child: KidSmallBtn(
                  label: '一键完成', color: KidColors.color3, onTap: onCompleteAll))
        ]),
      ]));
}

class _SubjectNavigator extends StatelessWidget {
  const _SubjectNavigator(
      {required this.groups,
      required this.selectedIndex,
      required this.onSelected});
  final List<TaskGroup> groups;
  final int selectedIndex;
  final ValueChanged<int> onSelected;
  @override
  Widget build(BuildContext context) => SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: Row(
          children: List.generate(groups.length, (i) {
        final g = groups[i];
        final isSel = i == selectedIndex;
        final color = g.subject.contains('语')
            ? KidColors.color5
            : (g.subject.contains('数') ? KidColors.color2 : KidColors.color4);
        return GestureDetector(
            onTap: () => onSelected(i),
            child: Container(
                margin: const EdgeInsets.only(right: 12),
                padding:
                    const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
                decoration: BoxDecoration(
                    color: isSel ? color : KidColors.white,
                    borderRadius: BorderRadius.circular(20),
                    border: Border.all(
                        color: isSel ? color : KidColors.black, width: 2)),
                child: Row(children: [
                  if (g.status == 'completed')
                    const Icon(Icons.check_circle_rounded,
                        color: KidColors.color3, size: 18),
                  const SizedBox(width: 4),
                  Text(g.subject,
                      style: TextStyle(
                          fontWeight: FontWeight.w900,
                          color: isSel ? KidColors.white : KidColors.black)),
                  const SizedBox(width: 8),
                  Text('${g.completed}/${g.total}',
                      style: TextStyle(
                          fontWeight: FontWeight.w900,
                          color: isSel ? KidColors.white : KidColors.black))
                ])));
      })));
}

class _FocusedSubjectStage extends StatelessWidget {
  const _FocusedSubjectStage(
      {required this.group,
      required this.homeworkGroups,
      required this.tasks,
      required this.busy,
      required this.onToggleTask});
  final TaskGroup group;
  final List<HomeworkGroup> homeworkGroups;
  final List<TaskItem> tasks;
  final bool busy;
  final Future<void> Function(TaskItem, bool) onToggleTask;
  @override
  Widget build(BuildContext context) => KidCard(
      borderColor: KidColors.black,
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text('${group.subject} 挑战舞台',
            style: const TextStyle(fontSize: 24, fontWeight: FontWeight.w900)),
        const SizedBox(height: 20),
        ...homeworkGroups.map(
            (hw) =>
                Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Padding(
                      padding: const EdgeInsets.symmetric(vertical: 8),
                      child: Text(hw.groupTitle,
                          style: const TextStyle(
                              fontSize: 18, fontWeight: FontWeight.w800))),
                  ...tasks.where((t) => t.groupTitle == hw.groupTitle).map(
                      (t) => Container(
                          margin: const EdgeInsets.only(bottom: 8),
                          decoration: BoxDecoration(
                              color: t.completed
                                  ? KidColors.color3.withAlpha(30)
                                  : Colors.white,
                              borderRadius: BorderRadius.circular(16),
                              border: Border.all(
                                  color: t.completed
                                      ? KidColors.color3
                                      : KidColors.black,
                                  width: 2)),
                          child: CheckboxListTile(
                              value: t.completed,
                              onChanged: (v) => onToggleTask(t, v ?? false),
                              activeColor: KidColors.color3,
                              title: Text(t.content,
                                  style: TextStyle(
                                      fontWeight: FontWeight.w700,
                                      decoration: t.completed
                                          ? TextDecoration.lineThrough
                                          : null))))),
                ])),
      ]));
}

class _WordPlaybackPanel extends StatelessWidget {
  const _WordPlaybackPanel(
      {required this.state,
      required this.onLanguageChanged,
      required this.onModeChanged,
      required this.onSyncBackend,
      required this.onPlayCurrent,
      required this.onReplayCurrent,
      required this.onNextWord,
      required this.onPreviousWord,
      required this.onPeekChanged,
      required this.onSubmitPhoto});
  final WordPlaybackState state;
  final ValueChanged<WordPlaybackLanguage> onLanguageChanged;
  final ValueChanged<WordPlaybackMode> onModeChanged;
  final ValueChanged<bool> onPeekChanged;
  final VoidCallback onSyncBackend,
      onReplayCurrent,
      onNextWord,
      onPreviousWord,
      onSubmitPhoto;
  final Future<void> Function() onPlayCurrent;
  String get _instructionText {
    if (!state.hasWords) return '请同步词单';
    if (state.isPeeking) return '预览模式';

    if (state.language == WordPlaybackLanguage.english) {
      return state.mode == WordPlaybackMode.word
          ? '听到单词，请默写英文'
          : '听到释义，请默写英文单词';
    } else {
      return state.mode == WordPlaybackMode.word
          ? '听到词语，请默写中文'
          : '听到释义，请默写中文词语';
    }
  }

  String get _submitLabel {
    if (state.isBusy) return '上传中';
    if (state.session?.isGradingPending == true) return '后台批改中';
    if (state.session?.hasGradingResult == true) return '重新拍照交卷';
    return '📸 拍照交卷';
  }

  @override
  Widget build(BuildContext context) {
    return KidCard(
        borderColor: KidColors.black,
        padding: const EdgeInsets.all(32),
        child: Column(children: [
          Row(children: [
            const Text('听写舞台',
                style: TextStyle(fontSize: 26, fontWeight: FontWeight.w900)),
            const Spacer(),
            GestureDetector(
                onLongPressStart: (_) => onPeekChanged(true),
                onLongPressEnd: (_) => onPeekChanged(false),
                child: Container(
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(
                        color: state.isPeeking
                            ? KidColors.color5
                            : KidColors.white,
                        shape: BoxShape.circle,
                        border: Border.all(color: KidColors.black, width: 2)),
                    child: Icon(
                        state.isPeeking
                            ? Icons.visibility_rounded
                            : Icons.visibility_off_rounded,
                        color:
                            state.isPeeking ? Colors.white : KidColors.black))),
            const SizedBox(width: 12),
            SizedBox(
                width: 140,
                child: KidSmallBtn(
                    label: state.isBusy ? '中...' : '同步云端',
                    color: KidColors.color2,
                    onTap: state.isBusy ? null : onSyncBackend)),
          ]),
          const SizedBox(height: 24),
          Wrap(spacing: 8, runSpacing: 8, children: [
            _TabBtn(
                label: '英语',
                isSel: state.language == WordPlaybackLanguage.english,
                onTap: () => onLanguageChanged(WordPlaybackLanguage.english)),
            _TabBtn(
                label: '语文',
                isSel: state.language == WordPlaybackLanguage.chinese,
                onTap: () => onLanguageChanged(WordPlaybackLanguage.chinese)),
            const SizedBox(width: 8),
            _TabBtn(
                label: '听词',
                isSel: state.mode == WordPlaybackMode.word,
                onTap: () => onModeChanged(WordPlaybackMode.word)),
            _TabBtn(
                label: '听义',
                isSel: state.mode == WordPlaybackMode.meaning,
                onTap: () => onModeChanged(WordPlaybackMode.meaning))
          ]),
          const SizedBox(height: 32),
          Container(
              width: double.infinity,
              padding: const EdgeInsets.symmetric(vertical: 80, horizontal: 32),
              decoration: BoxDecoration(
                  color: KidColors.color4,
                  borderRadius: BorderRadius.circular(32),
                  border: Border.all(color: KidColors.black, width: 3)),
              child: Column(children: [
                Text(
                    state.hasWords
                        ? (state.isPeeking
                            ? state.currentWord
                            : '挑战 #${state.currentDisplayIndex}')
                        : '等待中',
                    textAlign: TextAlign.center,
                    style: const TextStyle(
                        fontSize: 64,
                        fontWeight: FontWeight.w900,
                        color: KidColors.black)),
                const SizedBox(height: 16),
                Text(_instructionText,
                    style: TextStyle(
                        fontSize: 18,
                        fontWeight: FontWeight.w700,
                        color: KidColors.black.withAlpha(120))),
                const SizedBox(height: 40),
                LayoutBuilder(builder: (context, constraints) {
                  return Stack(children: [
                    Container(
                        height: 16,
                        decoration: BoxDecoration(
                            color: KidColors.black.withAlpha(30),
                            borderRadius: BorderRadius.circular(8))),
                    AnimatedContainer(
                        duration: const Duration(milliseconds: 400),
                        height: 16,
                        width: constraints.maxWidth * state.progress,
                        decoration: BoxDecoration(
                            color: KidColors.black,
                            borderRadius: BorderRadius.circular(8))),
                  ]);
                }),
                const SizedBox(height: 16),
                Text('进度 ${state.currentDisplayIndex} / ${state.totalWords}',
                    style: const TextStyle(
                        fontWeight: FontWeight.w900, fontSize: 18)),
              ])),
          if (state.noticeMessage != null || state.errorMessage != null) ...[
            const SizedBox(height: 24),
            Container(
                padding: const EdgeInsets.all(20),
                width: double.infinity,
                decoration: BoxDecoration(
                    color: state.errorMessage != null
                        ? KidColors.color5.withAlpha(40)
                        : KidColors.color3.withAlpha(40),
                    borderRadius: BorderRadius.circular(24),
                    border: Border.all(
                        color: state.errorMessage != null
                            ? KidColors.color5
                            : KidColors.color3,
                        width: 2)),
                child: Row(children: [
                  Icon(
                      state.errorMessage != null
                          ? Icons.error_outline_rounded
                          : Icons.tips_and_updates_rounded,
                      color: state.errorMessage != null
                          ? KidColors.color5
                          : KidColors.color3),
                  const SizedBox(width: 12),
                  Expanded(
                      child: Text(state.errorMessage ?? state.noticeMessage!,
                          style: TextStyle(
                              fontWeight: FontWeight.w800,
                              color: state.errorMessage != null
                                  ? KidColors.color5
                                  : KidColors.color3.withAlpha(200),
                              fontSize: 16)))
                ]))
          ],
          if (state.lastSubmission != null) ...[
            const SizedBox(height: 24),
            _LastSubmissionPreviewCard(
              snapshot: state.lastSubmission!,
              session: state.session,
            ),
          ],
          if (state.session?.gradingStatus.isNotEmpty == true &&
              state.session?.gradingStatus != 'idle') ...[
            const SizedBox(height: 24),
            _GradingJourneyCard(session: state.session!),
            const SizedBox(height: 16),
            _GradingStatusCard(session: state.session!),
          ],
          const SizedBox(height: 32),
          Row(children: [
            Expanded(
                child: KidSmallBtn(
                    label: '上一个',
                    color: KidColors.color1,
                    onTap: state.canPrevious && !state.isBusy
                        ? onPreviousWord
                        : null)),
            const SizedBox(width: 12),
            Expanded(
                flex: 2,
                child: KidSmallBtn(
                    label: state.isSpeaking ? '播报中' : '开始播报',
                    color: KidColors.color3,
                    onTap: state.hasWords && !state.isSpeaking && !state.isBusy
                        ? onPlayCurrent
                        : null)),
            const SizedBox(width: 12),
            Expanded(
                child: KidSmallBtn(
                    label: state.canNext ? '下一个' : '已播完',
                    color: KidColors.color1,
                    onTap: state.canNext && !state.isBusy ? onNextWord : null))
          ]),
          const SizedBox(height: 16),
          Row(children: [
            Expanded(
                child: KidSmallBtn(
                    label: '重播',
                    color: KidColors.color5,
                    onTap: state.hasWords && !state.isBusy
                        ? onReplayCurrent
                        : null)),
            const SizedBox(width: 12),
            Expanded(
                child: KidSmallBtn(
                    label: _submitLabel,
                    color: KidColors.color5,
                    onTap: state.hasWords &&
                            !state.isBusy &&
                            !(state.session?.isGradingPending ?? false)
                        ? onSubmitPhoto
                        : null)),
          ]),
        ]));
  }
}

class _LastSubmissionPreviewCard extends StatelessWidget {
  const _LastSubmissionPreviewCard({
    required this.snapshot,
    required this.session,
  });

  final DictationSubmissionSnapshot snapshot;
  final DictationSession? session;

  @override
  Widget build(BuildContext context) {
    final stageMeta = describeDictationStage(session);
    final photoToken = _shortTraceToken(session?.debugContext?.photoSha1);

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: KidColors.color4.withAlpha(18),
        borderRadius: BorderRadius.circular(24),
        border: Border.all(color: KidColors.color4, width: 2),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            '最近一次交卷',
            style: TextStyle(fontSize: 18, fontWeight: FontWeight.w900),
          ),
          const SizedBox(height: 8),
          Text(
            '先看看照片是否清楚，再等 AI 给结果。',
            style: TextStyle(
              fontSize: 15,
              fontWeight: FontWeight.w800,
              color: KidColors.black.withAlpha(170),
            ),
          ),
          const SizedBox(height: 16),
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                width: 120,
                height: 120,
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(22),
                  border: Border.all(color: KidColors.black, width: 2),
                ),
                child: ClipRRect(
                  borderRadius: BorderRadius.circular(20),
                  child: snapshot.previewBytes != null
                      ? Image.memory(
                          snapshot.previewBytes!,
                          fit: BoxFit.cover,
                          gaplessPlayback: true,
                        )
                      : const Icon(
                          Icons.photo_camera_back_rounded,
                          size: 44,
                          color: KidColors.black,
                        ),
                ),
              ),
              const SizedBox(width: 16),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Wrap(
                      spacing: 8,
                      runSpacing: 8,
                      children: [
                        _MiniTraceChip(
                            label:
                                _formatSubmissionClock(snapshot.submittedAt)),
                        _MiniTraceChip(
                            label: _formatSubmissionBytes(snapshot.byteCount)),
                        _MiniTraceChip(label: stageMeta.label),
                        if (photoToken.isNotEmpty)
                          _MiniTraceChip(label: '图片 $photoToken'),
                      ],
                    ),
                    const SizedBox(height: 14),
                    Text(
                      stageMeta.hint,
                      style: TextStyle(
                        fontSize: 14,
                        fontWeight: FontWeight.w700,
                        color: KidColors.black.withAlpha(165),
                      ),
                    ),
                    const SizedBox(height: 12),
                    Text(
                      '如果模糊、歪斜或没拍全，现在就重拍一张。',
                      style: TextStyle(
                        fontSize: 13,
                        fontWeight: FontWeight.w800,
                        color: KidColors.color5.withAlpha(210),
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _GradingJourneyCard extends StatelessWidget {
  const _GradingJourneyCard({required this.session});

  final DictationSession session;

  @override
  Widget build(BuildContext context) {
    final stageMeta = describeDictationStage(session);
    final tracePills = _buildTracePills(session);
    final steps = _buildJourneySteps(session);

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: KidColors.color2.withAlpha(20),
        borderRadius: BorderRadius.circular(24),
        border: Border.all(color: KidColors.color2, width: 2),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            '交卷进度',
            style: TextStyle(fontSize: 18, fontWeight: FontWeight.w900),
          ),
          const SizedBox(height: 8),
          Text(
            '${stageMeta.label} · ${stageMeta.hint}',
            style: TextStyle(
              fontSize: 15,
              fontWeight: FontWeight.w800,
              color: KidColors.color1.withAlpha(210),
            ),
          ),
          const SizedBox(height: 16),
          ...steps.map((step) => Padding(
                padding: const EdgeInsets.only(bottom: 10),
                child: _JourneyStepCard(step: step),
              )),
          if (tracePills.isNotEmpty) ...[
            const SizedBox(height: 4),
            Text(
              '同步线索',
              style: TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w900,
                color: KidColors.black.withAlpha(160),
              ),
            ),
            const SizedBox(height: 10),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: tracePills
                  .map((item) => _MiniTraceChip(label: item))
                  .toList(),
            ),
          ],
        ],
      ),
    );
  }
}

class _JourneyStepCard extends StatelessWidget {
  const _JourneyStepCard({required this.step});

  final _JourneyStepData step;

  @override
  Widget build(BuildContext context) {
    Color accent;
    IconData icon;
    switch (step.state) {
      case _JourneyStepVisualState.complete:
        accent = KidColors.color3;
        icon = Icons.check_circle_rounded;
        break;
      case _JourneyStepVisualState.active:
        accent = KidColors.color4;
        icon = Icons.autorenew_rounded;
        break;
      case _JourneyStepVisualState.failed:
        accent = KidColors.color5;
        icon = Icons.error_outline_rounded;
        break;
      case _JourneyStepVisualState.pending:
        accent = KidColors.black.withAlpha(120);
        icon = Icons.radio_button_unchecked_rounded;
        break;
    }

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: accent, width: 2),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 38,
            height: 38,
            decoration: BoxDecoration(
              color: accent.withAlpha(35),
              shape: BoxShape.circle,
            ),
            child: Icon(icon, color: accent),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  step.title,
                  style: const TextStyle(
                      fontSize: 16, fontWeight: FontWeight.w900),
                ),
                const SizedBox(height: 4),
                Text(
                  step.caption,
                  style: TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w700,
                    color: KidColors.black.withAlpha(170),
                  ),
                ),
              ],
            ),
          ),
          if (step.timeLabel != null)
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
              decoration: BoxDecoration(
                color: accent.withAlpha(28),
                borderRadius: BorderRadius.circular(999),
              ),
              child: Text(
                step.timeLabel!,
                style: TextStyle(
                  fontWeight: FontWeight.w900,
                  color: accent,
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _MiniTraceChip extends StatelessWidget {
  const _MiniTraceChip({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: KidColors.black, width: 1.5),
      ),
      child: Text(
        label,
        style: const TextStyle(fontWeight: FontWeight.w900, fontSize: 13),
      ),
    );
  }
}

class _GradingStatusCard extends StatelessWidget {
  const _GradingStatusCard({required this.session});

  final DictationSession session;

  @override
  Widget build(BuildContext context) {
    final result = session.gradingResult;
    final stageMeta = describeDictationStage(session);
    final incorrectItems = result?.gradedItems
            .where((item) => !item.isCorrect || item.needsCorrection)
            .toList() ??
        const <DictationGradedItem>[];

    Color borderColor;
    Color backgroundColor;
    IconData icon;
    String title;
    String subtitle;

    switch (session.gradingStatus) {
      case 'pending':
      case 'processing':
        borderColor = KidColors.color2;
        backgroundColor = KidColors.color2.withAlpha(35);
        icon = Icons.hourglass_top_rounded;
        title = '交卷进行中 · ${stageMeta.label}';
        subtitle = stageMeta.hint;
        break;
      case 'failed':
        borderColor = KidColors.color5;
        backgroundColor = KidColors.color5.withAlpha(35);
        icon = Icons.error_outline_rounded;
        title = '这次受阻了';
        subtitle = session.gradingError?.isNotEmpty == true
            ? session.gradingError!
            : stageMeta.hint;
        break;
      case 'completed':
        borderColor = KidColors.color3;
        backgroundColor = KidColors.color3.withAlpha(35);
        icon = Icons.task_alt_rounded;
        title = result == null ? '批改已完成' : '批改完成 · ${result.score} 分';
        subtitle = result?.aiFeedback.isNotEmpty == true
            ? result!.aiFeedback
            : stageMeta.hint;
        break;
      default:
        borderColor = KidColors.black;
        backgroundColor = KidColors.white;
        icon = Icons.info_outline_rounded;
        title = '批改状态';
        subtitle = '等待后台状态同步。';
    }

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: backgroundColor,
        borderRadius: BorderRadius.circular(24),
        border: Border.all(color: borderColor, width: 2),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(icon, color: borderColor),
              const SizedBox(width: 12),
              Expanded(
                  child: Text(title,
                      style: const TextStyle(
                          fontSize: 18, fontWeight: FontWeight.w900))),
            ],
          ),
          const SizedBox(height: 8),
          Text(subtitle,
              style:
                  const TextStyle(fontSize: 15, fontWeight: FontWeight.w700)),
          const SizedBox(height: 12),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              _MiniTraceChip(label: '会话 ${session.sessionId}'),
              if (session.gradingRequestedAt?.isNotEmpty == true)
                _MiniTraceChip(
                    label:
                        '提交 ${_formatJourneyTime(session.gradingRequestedAt)}'),
              if (session.gradingCompletedAt?.isNotEmpty == true)
                _MiniTraceChip(
                    label:
                        '返回 ${_formatJourneyTime(session.gradingCompletedAt)}'),
            ],
          ),
          if (result != null) ...[
            const SizedBox(height: 12),
            Text('错题 ${incorrectItems.length} / ${result.gradedItems.length}',
                style: const TextStyle(fontWeight: FontWeight.w900)),
          ],
          if (incorrectItems.isNotEmpty) ...[
            const SizedBox(height: 12),
            ...incorrectItems.take(4).map((item) => Padding(
                  padding: const EdgeInsets.only(bottom: 8),
                  child: Text(
                    '#${item.index} ${item.expected} -> ${item.actual.isEmpty ? "未识别" : item.actual}${(item.comment?.isNotEmpty ?? false) ? " · ${item.comment}" : ""}',
                    style: const TextStyle(fontWeight: FontWeight.w700),
                  ),
                )),
          ],
        ],
      ),
    );
  }
}

class _HomeModeSwitcher extends StatelessWidget {
  const _HomeModeSwitcher({required this.selectedTab, required this.onChanged});
  final _PadHomeTab selectedTab;
  final ValueChanged<_PadHomeTab> onChanged;
  @override
  Widget build(BuildContext context) => Row(children: [
        Expanded(
            child: _TabBtn(
                label: '今日挑战',
                isSel: selectedTab == _PadHomeTab.tasks,
                onTap: () => onChanged(_PadHomeTab.tasks))),
        const SizedBox(width: 12),
        Expanded(
            child: _TabBtn(
                label: '听写练词',
                isSel: selectedTab == _PadHomeTab.words,
                onTap: () => onChanged(_PadHomeTab.words)))
      ]);
}

class _TabBtn extends StatelessWidget {
  const _TabBtn(
      {required this.label, required this.isSel, required this.onTap});
  final String label;
  final bool isSel;
  final VoidCallback onTap;
  @override
  Widget build(BuildContext context) => GestureDetector(
      onTap: onTap,
      child: Container(
          padding: const EdgeInsets.symmetric(vertical: 12),
          alignment: Alignment.center,
          decoration: BoxDecoration(
              color: isSel ? KidColors.black : KidColors.white,
              borderRadius: BorderRadius.circular(16),
              border: Border.all(color: KidColors.black, width: 2)),
          child: Text(label,
              style: TextStyle(
                  color: isSel ? KidColors.white : KidColors.black,
                  fontWeight: FontWeight.w900))));
}

class _EmptyBoard extends StatelessWidget {
  const _EmptyBoard({required this.title, required this.description});
  final String title, description;
  @override
  Widget build(BuildContext context) => Center(
          child: Column(children: [
        const Icon(Icons.coffee_rounded, size: 64, color: KidColors.color2),
        const SizedBox(height: 16),
        Text(title,
            style: const TextStyle(fontSize: 22, fontWeight: FontWeight.w900)),
        Text(description)
      ]));
}

class _BannerCard extends StatelessWidget {
  const _BannerCard({required this.message});
  final String message;
  @override
  Widget build(BuildContext context) => Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
          color: KidColors.color5, borderRadius: BorderRadius.circular(12)),
      child: Row(children: [
        const Icon(Icons.error_outline_rounded, color: Colors.white),
        const SizedBox(width: 12),
        Expanded(
            child: Text(message,
                style: const TextStyle(
                    color: Colors.white, fontWeight: FontWeight.w700)))
      ]));
}
