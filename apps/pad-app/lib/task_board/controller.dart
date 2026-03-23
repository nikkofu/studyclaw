import 'package:flutter/foundation.dart';
import 'package:pad_app/task_board/completion_encouragement.dart';
import 'package:pad_app/task_board/feedback.dart';
import 'package:pad_app/task_board/models.dart';
import 'package:pad_app/task_board/repository.dart';
import 'package:pad_app/task_board/daily_stats.dart';

const bool _hotTaskLaunchV1 = bool.fromEnvironment('hot_task_launch_v1', defaultValue: false);
const bool _hotTaskResumeV1 = bool.fromEnvironment('hot_task_resume_v1', defaultValue: false);
const bool _hotTaskRewardsV1 = bool.fromEnvironment('hot_task_rewards_v1', defaultValue: false);

enum TaskBoardScreenStatus { loading, empty, error, success }

enum TaskBoardNoticeTone { success, info }

const Object _missing = Object();

class TaskBoardViewState {
  const TaskBoardViewState({
    required this.status,
    this.board,
    this.dailyStats,
    this.errorMessage,
    this.noticeMessage,
    this.noticeTone = TaskBoardNoticeTone.success,
    this.isRefreshing = false,
    this.isUpdating = false,
    this.hasLoadedOnce = false,
    this.lastSyncedAt,
  });

  factory TaskBoardViewState.initial() {
    return const TaskBoardViewState(status: TaskBoardScreenStatus.empty);
  }

  final TaskBoardScreenStatus status;
  final TaskBoard? board;
  final DailyStats? dailyStats;
  final String? errorMessage;
  final String? noticeMessage;
  final TaskBoardNoticeTone noticeTone;
  final bool isRefreshing;
  final bool isUpdating;
  final bool hasLoadedOnce;
  final DateTime? lastSyncedAt;

  bool get isBusy {
    return status == TaskBoardScreenStatus.loading ||
        isRefreshing ||
        isUpdating;
  }

  String? get activityLabel {
    if (status == TaskBoardScreenStatus.loading) {
      return '正在加载任务板...';
    }
    if (isRefreshing) {
      return '正在刷新任务板...';
    }
    if (isUpdating) {
      return '正在同步任务状态...';
    }
    return null;
  }

  TaskBoardViewState copyWith({
    TaskBoardScreenStatus? status,
    Object? board = _missing,
    Object? dailyStats = _missing,
    Object? errorMessage = _missing,
    Object? noticeMessage = _missing,
    Object? noticeTone = _missing,
    bool? isRefreshing,
    bool? isUpdating,
    bool? hasLoadedOnce,
    Object? lastSyncedAt = _missing,
  }) {
    return TaskBoardViewState(
      status: status ?? this.status,
      board: board == _missing ? this.board : board as TaskBoard?,
      dailyStats:
          dailyStats == _missing ? this.dailyStats : dailyStats as DailyStats?,
      errorMessage: errorMessage == _missing
          ? this.errorMessage
          : errorMessage as String?,
      noticeMessage: noticeMessage == _missing
          ? this.noticeMessage
          : noticeMessage as String?,
      noticeTone: noticeTone == _missing
          ? this.noticeTone
          : noticeTone as TaskBoardNoticeTone,
      isRefreshing: isRefreshing ?? this.isRefreshing,
      isUpdating: isUpdating ?? this.isUpdating,
      hasLoadedOnce: hasLoadedOnce ?? this.hasLoadedOnce,
      lastSyncedAt: lastSyncedAt == _missing
          ? this.lastSyncedAt
          : lastSyncedAt as DateTime?,
    );
  }
}

class TaskBoardController extends ChangeNotifier {
  TaskBoardController({required TaskBoardRepository repository})
      : _repository = repository;

  final TaskBoardRepository _repository;

  @visibleForTesting
  static Map<String, bool> hotTaskFlags() {
    return const {
      'hot_task_launch_v1': _hotTaskLaunchV1,
      'hot_task_resume_v1': _hotTaskResumeV1,
      'hot_task_rewards_v1': _hotTaskRewardsV1,
    };
  }

  TaskItem? resolveLaunchTask(TaskBoard board) {
    final recommendedId = board.launchRecommendation?.itemId;
    if (recommendedId != null) {
      for (final task in board.tasks) {
        if (task.taskId == recommendedId && !task.completed) {
          return task;
        }
      }
    }
    for (final task in board.tasks) {
      if (!task.completed) {
        return task;
      }
    }
    return null;
  }

  bool get hotTaskLaunchEnabled => _hotTaskLaunchV1;

  TaskBoardViewState _state = TaskBoardViewState.initial();
  bool _showCompletedHistory = false;
  Set<String> _expandedHomeworkGroupKeys = <String>{};

  TaskBoardViewState get state => _state;
  bool get showCompletedHistory => _showCompletedHistory;
  Set<String> get expandedHomeworkGroupKeys => _expandedHomeworkGroupKeys;

  void toggleCompletedHistory(bool value) {
    if (_showCompletedHistory == value) {
      return;
    }
    _showCompletedHistory = value;
    notifyListeners();
  }

  void toggleHomeworkGroupExpanded(
    String subject,
    String groupTitle,
    bool expanded,
  ) {
    final key = _groupKey(subject, groupTitle);
    final nextExpandedKeys = <String>{..._expandedHomeworkGroupKeys};

    if (expanded) {
      nextExpandedKeys.add(key);
    } else {
      nextExpandedKeys.remove(key);
    }

    _expandedHomeworkGroupKeys = nextExpandedKeys;
    notifyListeners();
  }

  void presentValidationError(String message) {
    final board = _state.board;
    _state = _state.copyWith(
      status:
          board == null ? TaskBoardScreenStatus.error : _statusForBoard(board),
      errorMessage: message,
      noticeMessage: null,
      noticeTone: TaskBoardNoticeTone.success,
      isRefreshing: false,
      isUpdating: false,
    );
    notifyListeners();
  }

  Future<void> loadBoard(
    TaskBoardRequest request, {
    bool showLoadingState = false,
    String? successMessage,
  }) async {
    final shouldReplaceContent = showLoadingState || _state.board == null;
    _state = _state.copyWith(
      status: shouldReplaceContent
          ? TaskBoardScreenStatus.loading
          : _statusForBoard(_state.board!),
      errorMessage: null,
      noticeMessage: null,
      noticeTone: TaskBoardNoticeTone.success,
      isRefreshing: !shouldReplaceContent,
      isUpdating: false,
    );
    notifyListeners();

    try {
      final results = await Future.wait([
        _repository.fetchBoard(request),
        _repository.fetchDailyStats(request),
      ]);
      final board = results[0] as TaskBoard;
      final dailyStats = results[1] as DailyStats;
      _applyBoard(board, dailyStats: dailyStats, noticeMessage: successMessage);
    } catch (error) {
      _handleError(error);
    }
  }

  Future<void> refresh(TaskBoardRequest request) {
    return loadBoard(
      request,
      showLoadingState: _state.board == null,
      successMessage: '任务板已刷新好，继续今天的挑战吧。',
    );
  }

  Future<void> updateSingleTask(
    TaskBoardRequest request,
    TaskItem task,
    bool completed,
  ) async {
    await _runMutation(
      request,
      () => _repository.updateSingleTask(
        request,
        taskId: task.taskId,
        completed: completed,
      ),
      completed ? '这一步完成啦，继续向前。' : '这一步先放回待完成，我们重新来一次。',
      completionKind: completed ? TaskCompletionKind.singleTask : null,
      subject: task.subject,
      groupTitle: task.groupTitle,
      taskContent: task.content,
    );
  }

  Future<void> updateSubjectGroup(
    TaskBoardRequest request,
    TaskGroup group,
    bool completed,
  ) async {
    await _runMutation(
      request,
      () => _repository.updateTaskGroup(
        request,
        subject: group.subject,
        completed: completed,
      ),
      completed
          ? '${group.subject} 这一科完成啦，继续保持。'
          : '${group.subject} 这一科已放回待完成，我们再慢慢来。',
      completionKind: completed ? TaskCompletionKind.subjectGroup : null,
      subject: group.subject,
    );
  }

  Future<void> updateHomeworkGroup(
    TaskBoardRequest request,
    HomeworkGroup group,
    bool completed,
  ) async {
    await _runMutation(
      request,
      () => _repository.updateTaskGroup(
        request,
        subject: group.subject,
        groupTitle: group.groupTitle,
        completed: completed,
      ),
      completed
          ? '“${group.groupTitle}”这一组完成啦。'
          : '“${group.groupTitle}”这一组已放回待完成，我们再来一次。',
      completionKind: completed ? TaskCompletionKind.homeworkGroup : null,
      subject: group.subject,
      groupTitle: group.groupTitle,
    );
  }

  Future<void> updateAllTasks(
    TaskBoardRequest request, {
    required bool completed,
  }) async {
    await _runMutation(
      request,
      () => _repository.updateAllTasks(request, completed: completed),
      completed ? '今天的挑战全部完成啦！' : '今天的任务已重置好，我们重新出发。',
      completionKind: completed ? TaskCompletionKind.allTasks : null,
    );
  }

  Future<void> _runMutation(
    TaskBoardRequest request,
    Future<TaskBoard> Function() action,
    String successMessage, {
    TaskCompletionKind? completionKind,
    String? subject,
    String? groupTitle,
    String? taskContent,
  }) async {
    final previousBoard = _state.board;
    _state = _state.copyWith(
      errorMessage: null,
      noticeMessage: null,
      noticeTone: TaskBoardNoticeTone.success,
      isUpdating: true,
      isRefreshing: false,
    );
    notifyListeners();

    try {
      final board = await action();
      // After mutation, refresh daily stats too
      final dailyStats = await _repository.fetchDailyStats(request);
      final resolvedMessage = completionKind == null
          ? successMessage
          : buildTaskCompletionEncouragement(
                  kind: completionKind,
                  previousBoard: previousBoard,
                  board: board,
                  dailyStats: dailyStats,
                  subject: subject,
                  groupTitle: groupTitle,
                  taskContent: taskContent) ??
              successMessage;
      _applyBoard(
        board,
        dailyStats: dailyStats,
        noticeMessage: resolvedMessage,
      );
    } catch (error) {
      _handleError(error);
    }
  }

  void _applyBoard(TaskBoard board,
      {DailyStats? dailyStats, String? noticeMessage}) {
    _expandedHomeworkGroupKeys = board.homeworkGroups
        .where((group) => group.status != 'completed')
        .map((group) => _groupKey(group.subject, group.groupTitle))
        .toSet();

    if (!board.tasks.any((task) => task.completed)) {
      _showCompletedHistory = false;
    }

    _state = _state.copyWith(
      status: _statusForBoard(board),
      board: board,
      dailyStats: dailyStats,
      errorMessage: null,
      noticeMessage: _resolveNoticeMessage(board, dailyStats, noticeMessage),
      noticeTone: TaskBoardNoticeTone.success,
      isRefreshing: false,
      isUpdating: false,
      hasLoadedOnce: true,
      lastSyncedAt: DateTime.now(),
    );
    notifyListeners();
  }

  void _handleError(Object error) {
    final board = _state.board;
    final feedback = describePadApiFeedback(error);
    _state = _state.copyWith(
      status:
          board == null ? TaskBoardScreenStatus.error : _statusForBoard(board),
      errorMessage: feedback.isNotice ? null : feedback.message,
      noticeMessage: feedback.isNotice ? feedback.message : null,
      noticeTone: feedback.kind == PadApiFeedbackKind.infoNotice
          ? TaskBoardNoticeTone.info
          : TaskBoardNoticeTone.success,
      isRefreshing: false,
      isUpdating: false,
      hasLoadedOnce: true,
    );
    notifyListeners();
  }

  TaskBoardScreenStatus _statusForBoard(TaskBoard board) {
    return board.tasks.isEmpty
        ? TaskBoardScreenStatus.empty
        : TaskBoardScreenStatus.success;
  }

  String? _resolveNoticeMessage(
    TaskBoard board,
    DailyStats? dailyStats,
    String? noticeMessage,
  ) {
    if (noticeMessage != null && noticeMessage.trim().isNotEmpty) {
      return noticeMessage;
    }
    final encouragement = dailyStats?.encouragement.trim() ?? '';
    if (encouragement.isNotEmpty) {
      return encouragement;
    }
    final boardMessage = board.message?.trim();
    if (boardMessage == null || boardMessage.isEmpty) {
      return null;
    }
    if (boardMessage.toUpperCase() == 'OK') {
      return null;
    }
    return boardMessage;
  }

  String _groupKey(String subject, String groupTitle) {
    return '$subject::$groupTitle';
  }
}
