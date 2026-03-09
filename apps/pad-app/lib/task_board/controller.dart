import 'package:flutter/foundation.dart';
import 'package:pad_app/task_board/api_client.dart';
import 'package:pad_app/task_board/models.dart';
import 'package:pad_app/task_board/repository.dart';

enum TaskBoardScreenStatus { loading, empty, error, success }

enum TaskBoardNoticeTone { success, info }

const Object _missing = Object();

class TaskBoardViewState {
  const TaskBoardViewState({
    required this.status,
    this.board,
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
      final board = await _repository.fetchBoard(request);
      _applyBoard(board, noticeMessage: successMessage);
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
      () => _repository.updateAllTasks(request, completed: completed),
      completed ? '已将全部任务同步为完成' : '已将全部任务恢复为待完成',
    );
  }

  Future<void> _runMutation(
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
      _applyBoard(board, noticeMessage: successMessage);
    } catch (error) {
      _handleError(error);
    }
  }

  void _applyBoard(TaskBoard board, {String? noticeMessage}) {
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
    final feedback = _describeFeedback(error);
    _state = _state.copyWith(
      status:
          board == null ? TaskBoardScreenStatus.error : _statusForBoard(board),
      errorMessage: feedback.isNotice ? null : feedback.message,
      noticeMessage: feedback.isNotice ? feedback.message : null,
      noticeTone: feedback.noticeTone,
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

  _TaskBoardFeedback _describeFeedback(Object error) {
    if (error is TaskApiException) {
      if (error.errorCode == 'status_unchanged') {
        return _TaskBoardFeedback.notice(
          _describeStatusUnchanged(error),
          noticeTone: TaskBoardNoticeTone.info,
        );
      }
      return _TaskBoardFeedback.error(_describeTaskApiError(error));
    }
    if (error is FormatException) {
      return const _TaskBoardFeedback.error('服务端返回了无法解析的数据。');
    }
    return _TaskBoardFeedback.error('同步失败：$error');
  }

  String _describeTaskApiError(TaskApiException error) {
    final details = error.details ?? const <String, dynamic>{};

    switch (error.errorCode) {
      case 'task_not_found':
        final taskId = _detailInt(details, 'task_id');
        if (taskId != null) {
          return '任务 #$taskId 不存在，可能已被删除或日期已变更。';
        }
        return '当前日期没有可同步的任务，请先刷新任务板。';
      case 'task_group_not_found':
        final subject = _detailString(details, 'subject');
        final groupTitle = _detailString(details, 'group_title');
        if (subject != null && groupTitle != null) {
          return '没有找到“$subject / $groupTitle”对应的任务分组，请先刷新任务板。';
        }
        if (subject != null) {
          return '没有找到“$subject”学科下可同步的任务，请先刷新任务板。';
        }
        return '没有找到可同步的任务分组，请先刷新任务板。';
      case 'missing_required_fields':
        final fields = _detailStringList(details, 'fields');
        if (fields.isNotEmpty) {
          return '缺少必要参数：${fields.map(_fieldLabel).join('、')}。';
        }
        return '缺少必要参数，请检查同步配置后重试。';
      case 'invalid_request_fields':
        final fields = _detailStringList(details, 'fields');
        if (fields.isNotEmpty) {
          return '这些参数缺失或格式不正确：${fields.map(_fieldLabel).join('、')}。';
        }
        return '请求参数缺失或格式不正确，请检查后重试。';
      case 'invalid_query_parameter':
        final field = _detailString(details, 'field');
        if (field != null) {
          return '查询参数“${_fieldLabel(field)}”格式不正确。';
        }
        return '查询参数格式不正确，请检查后重试。';
      case 'invalid_date':
        final field = _detailString(details, 'field');
        if (field != null) {
          return '“${_fieldLabel(field)}”格式无效，请使用 YYYY-MM-DD。';
        }
        return '日期格式无效，请使用 YYYY-MM-DD。';
      case 'invalid_json':
        return '请求体格式无效，请重试。';
      case 'invalid_request':
        return '请求参数无效，请检查同步配置后重试。';
      case 'parser_unavailable':
        return '解析服务暂不可用，请稍后再试。';
      case 'tasks_not_extractable':
        return '任务内容暂时无法解析，请先回到家长端重新确认。';
      case 'internal_error':
        return '服务端处理失败，请稍后再试。';
    }

    if (error.statusCode > 0) {
      return '请求失败（${error.statusCode}）：${error.message}';
    }
    return '网络请求失败：${error.message}';
  }

  String _describeStatusUnchanged(TaskApiException error) {
    final details = error.details ?? const <String, dynamic>{};
    final statusLabel = _statusLabel(_detailString(details, 'status'));
    final taskId = _detailInt(details, 'task_id');
    final subject = _detailString(details, 'subject');
    final groupTitle = _detailString(details, 'group_title');

    if (taskId != null) {
      return '任务 #$taskId 已经是$statusLabel状态，无需重复同步。';
    }
    if (subject != null && groupTitle != null) {
      return '“$subject / $groupTitle”分组已经是$statusLabel状态，无需重复同步。';
    }
    if (subject != null) {
      return '“$subject”学科任务已经是$statusLabel状态，无需重复同步。';
    }
    return '全部任务已经是$statusLabel状态，无需重复同步。';
  }

  int? _detailInt(Map<String, dynamic> details, String key) {
    final value = details[key];
    if (value is int) {
      return value;
    }
    if (value is num) {
      return value.toInt();
    }
    return int.tryParse(value?.toString() ?? '');
  }

  String? _detailString(Map<String, dynamic> details, String key) {
    final value = details[key]?.toString().trim();
    if (value == null || value.isEmpty) {
      return null;
    }
    return value;
  }

  List<String> _detailStringList(Map<String, dynamic> details, String key) {
    final value = details[key];
    if (value is! List) {
      return const <String>[];
    }

    return value
        .map((item) => item?.toString().trim() ?? '')
        .where((item) => item.isNotEmpty)
        .toList();
  }

  String _fieldLabel(String field) {
    switch (field) {
      case 'subject':
        return '学科';
      case 'group_title':
        return '任务分组';
      case 'task_id':
        return '任务 ID';
      case 'assigned_date':
      case 'date':
        return '任务日期';
      case 'end_date':
        return '结束日期';
      case 'completed':
        return '完成状态';
      default:
        return field;
    }
  }

  String _statusLabel(String? status) {
    switch (status) {
      case 'completed':
        return '已完成';
      case 'pending':
        return '待完成';
      default:
        return '当前';
    }
  }

  String _groupKey(String subject, String groupTitle) {
    return '$subject::$groupTitle';
  }
}

class _TaskBoardFeedback {
  const _TaskBoardFeedback.error(this.message)
      : isNotice = false,
        noticeTone = TaskBoardNoticeTone.success;

  const _TaskBoardFeedback.notice(
    this.message, {
    this.noticeTone = TaskBoardNoticeTone.info,
  }) : isNotice = true;

  final String message;
  final bool isNotice;
  final TaskBoardNoticeTone noticeTone;
}
