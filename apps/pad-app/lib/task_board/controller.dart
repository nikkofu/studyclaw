import 'package:flutter/foundation.dart';
import 'package:pad_app/task_board/feedback.dart';
import 'package:pad_app/task_board/models.dart';
import 'package:pad_app/task_board/repository.dart';
import 'package:pad_app/task_board/daily_stats.dart';

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
      successMessage: '任务板已手动刷新',
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
      completed ? '已同步单个任务完成状态' : '已恢复单个任务为待完成',
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
          ? '已将 ${group.subject} 学科任务标记为完成'
          : '已将 ${group.subject} 学科任务恢复为待完成',
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
          ? '已将 ${group.groupTitle} 分组标记为完成'
          : '已将 ${group.groupTitle} 分组恢复为待完成',
    );
  }

  Future<void> updateAllTasks(
    TaskBoardRequest request, {
    required bool completed,
  }) async {
    await _runMutation(
      request,
      () => _repository.updateAllTasks(request, completed: completed),
      completed ? '已将全部任务同步为完成' : '已将全部任务恢复为待完成',
    );
  }

  Future<void> _runMutation(
    TaskBoardRequest request,
    Future<TaskBoard> Function() action,
    String successMessage,
  ) async {
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
      _applyBoard(board, dailyStats: dailyStats, noticeMessage: successMessage);
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
      noticeMessage: _resolveNoticeMessage(board, noticeMessage),
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

  String? _resolveNoticeMessage(TaskBoard board, String? noticeMessage) {
    if (noticeMessage != null && noticeMessage.trim().isNotEmpty) {
      return noticeMessage;
    }
    final boardMessage = board.message?.trim();
    if (boardMessage == null || boardMessage.isEmpty) {
      return null;
    }
    return boardMessage;
  }

  String _groupKey(String subject, String groupTitle) {
    return '$subject::$groupTitle';
  }
}
