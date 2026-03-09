import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pad_app/app.dart';
import 'package:pad_app/task_board/api_client.dart';
import 'package:pad_app/task_board/models.dart';
import 'package:pad_app/task_board/repository.dart';

void main() {
  group('PadTaskBoardPage', () {
    testWidgets('single task checkbox sync succeeds', (tester) async {
      _setLargeViewport(tester);

      final initialBoard = _buildBoard(
        tasks: const <_TaskSeed>[
          _TaskSeed(
            taskId: 1,
            subject: '数学',
            groupTitle: '口算练习',
            content: '完成第 1 页',
          ),
        ],
      );
      final updatedBoard = _buildBoard(
        tasks: const <_TaskSeed>[
          _TaskSeed(
            taskId: 1,
            subject: '数学',
            groupTitle: '口算练习',
            content: '完成第 1 页',
            completed: true,
          ),
        ],
      );
      final repository = _FakeTaskBoardRepository(
        onFetch: (_) async => initialBoard,
        onUpdateSingleTask: (_, taskId, completed) async => updatedBoard,
      );

      await _pumpLoadedBoard(tester, repository: repository);

      await tester.tap(find.byType(CheckboxListTile).first);
      await tester.pump();
      await tester.pumpAndSettle();

      expect(repository.singleTaskUpdates.length, 1);
      expect(repository.singleTaskUpdates.single.taskId, 1);
      expect(repository.singleTaskUpdates.single.completed, isTrue);
      expect(find.text('已同步单个任务完成状态'), findsOneWidget);
      expect(find.text('未完成任务已经清空'), findsOneWidget);
    });

    testWidgets('group completion sync succeeds', (tester) async {
      _setLargeViewport(tester);

      final initialBoard = _buildBoard(
        tasks: const <_TaskSeed>[
          _TaskSeed(
            taskId: 1,
            subject: '数学',
            groupTitle: '口算练习',
            content: '完成第 1 页',
          ),
          _TaskSeed(
            taskId: 2,
            subject: '数学',
            groupTitle: '口算练习',
            content: '完成第 2 页',
          ),
        ],
      );
      final updatedBoard = _buildBoard(
        tasks: const <_TaskSeed>[
          _TaskSeed(
            taskId: 1,
            subject: '数学',
            groupTitle: '口算练习',
            content: '完成第 1 页',
            completed: true,
          ),
          _TaskSeed(
            taskId: 2,
            subject: '数学',
            groupTitle: '口算练习',
            content: '完成第 2 页',
            completed: true,
          ),
        ],
      );
      final repository = _FakeTaskBoardRepository(
        onFetch: (_) async => initialBoard,
        onUpdateTaskGroup: (_, subject, groupTitle, completed) async =>
            updatedBoard,
      );

      await _pumpLoadedBoard(tester, repository: repository);

      await tester.tap(find.text('分组完成'));
      await tester.pump();
      await tester.pumpAndSettle();

      expect(repository.groupUpdates.length, 1);
      expect(repository.groupUpdates.single.subject, '数学');
      expect(repository.groupUpdates.single.groupTitle, '口算练习');
      expect(repository.groupUpdates.single.completed, isTrue);
      expect(find.text('已将 口算练习 分组标记为完成'), findsOneWidget);
      expect(find.text('今天已经完成 2 条任务。'), findsOneWidget);
    });

    testWidgets('bulk complete sync succeeds', (tester) async {
      _setLargeViewport(tester);

      final initialBoard = _buildBoard(
        tasks: const <_TaskSeed>[
          _TaskSeed(
            taskId: 1,
            subject: '数学',
            groupTitle: '口算练习',
            content: '完成第 1 页',
          ),
          _TaskSeed(
            taskId: 2,
            subject: '英语',
            groupTitle: '背单词',
            content: '复习 20 个单词',
          ),
        ],
      );
      final updatedBoard = _buildBoard(
        tasks: const <_TaskSeed>[
          _TaskSeed(
            taskId: 1,
            subject: '数学',
            groupTitle: '口算练习',
            content: '完成第 1 页',
            completed: true,
          ),
          _TaskSeed(
            taskId: 2,
            subject: '英语',
            groupTitle: '背单词',
            content: '复习 20 个单词',
            completed: true,
          ),
        ],
      );
      final repository = _FakeTaskBoardRepository(
        onFetch: (_) async => initialBoard,
        onUpdateAllTasks: (_, completed) async => updatedBoard,
      );

      await _pumpLoadedBoard(tester, repository: repository);

      await tester.tap(find.widgetWithText(FilledButton, '全部完成'));
      await tester.pump();
      await tester.pumpAndSettle();

      expect(repository.bulkUpdates.length, 1);
      expect(repository.bulkUpdates.single, isTrue);
      expect(find.text('已将全部任务同步为完成'), findsOneWidget);
      expect(find.text('未完成任务已经清空'), findsOneWidget);
    });

    testWidgets('404 task error shows friendly message', (tester) async {
      _setLargeViewport(tester);

      final initialBoard = _buildBoard(
        tasks: const <_TaskSeed>[
          _TaskSeed(
            taskId: 1,
            subject: '数学',
            groupTitle: '口算练习',
            content: '完成第 1 页',
          ),
        ],
      );
      final repository = _FakeTaskBoardRepository(
        onFetch: (_) async => initialBoard,
        onUpdateSingleTask: (_, taskId, completed) async {
          throw TaskApiException(
            message: 'Task not found',
            errorCode: 'task_not_found',
            details: const {'task_id': 1},
            uri: Uri.parse('http://localhost:8080/api/v1/tasks/status/item'),
            statusCode: 404,
          );
        },
      );

      await _pumpLoadedBoard(tester, repository: repository);

      await tester.tap(find.byType(CheckboxListTile).first);
      await tester.pump();
      await tester.pumpAndSettle();

      expect(find.text('任务 #1 不存在，可能已被删除或日期已变更。'), findsOneWidget);
      expect(find.text('数学'), findsOneWidget);
      expect(find.text('Task not found'), findsNothing);
    });

    testWidgets('409 unchanged status shows info hint instead of error', (
      tester,
    ) async {
      _setLargeViewport(tester);

      final completedBoard = _buildBoard(
        tasks: const <_TaskSeed>[
          _TaskSeed(
            taskId: 1,
            subject: '数学',
            groupTitle: '口算练习',
            content: '完成第 1 页',
            completed: true,
          ),
        ],
      );
      final repository = _FakeTaskBoardRepository(
        onFetch: (_) async => completedBoard,
        onUpdateAllTasks: (_, completed) async {
          throw TaskApiException(
            message: 'All tasks are already completed',
            errorCode: 'status_unchanged',
            details: const {'status': 'completed'},
            uri: Uri.parse('http://localhost:8080/api/v1/tasks/status/all'),
            statusCode: 409,
          );
        },
      );

      await _pumpLoadedBoard(tester, repository: repository);

      await tester.tap(find.widgetWithText(FilledButton, '全部完成'));
      await tester.pump();
      await tester.pumpAndSettle();

      expect(find.text('全部任务已经是已完成状态，无需重复同步。'), findsOneWidget);
      expect(find.textContaining('请求失败'), findsNothing);
      expect(find.text('未完成任务已经清空'), findsOneWidget);
    });

    testWidgets('shows loading then empty state for an empty board', (
      tester,
    ) async {
      final completer = Completer<TaskBoard>();
      final repository = _FakeTaskBoardRepository(
        onFetch: (_) => completer.future,
        fallbackBoard: _emptyBoard(date: '2026-03-06'),
      );

      await tester.pumpWidget(
        StudyClawPadApp(
          autoLoad: true,
          initialDate: '2026-03-06',
          repository: repository,
        ),
      );

      await tester.pump();
      expect(find.text('正在加载任务板'), findsOneWidget);

      completer.complete(_emptyBoard(date: '2026-03-06'));
      await tester.pumpAndSettle();

      expect(find.text('当前日期没有任务'), findsOneWidget);
      expect(find.text('空任务板'), findsWidgets);
      expect(repository.fetchRequests.single.date, '2026-03-06');
    });

    testWidgets('shows error state and retry recovers', (tester) async {
      _setLargeViewport(tester);

      var attempts = 0;
      final repository = _FakeTaskBoardRepository(
        onFetch: (request) async {
          attempts += 1;
          if (attempts == 1) {
            throw TaskApiException(
              message: '服务暂时不可用',
              uri: Uri.parse('http://localhost:8080/api/v1/tasks'),
              statusCode: 500,
            );
          }
          return _boardWithTasks(date: request.date);
        },
      );

      await tester.pumpWidget(
        StudyClawPadApp(
          autoLoad: true,
          initialDate: '2026-03-06',
          repository: repository,
        ),
      );

      await tester.pump();
      await tester.pumpAndSettle();

      expect(find.text('加载失败'), findsOneWidget);
      expect(find.textContaining('请求失败（500）'), findsWidgets);

      await tester.tap(find.text('重试加载'));
      await tester.pump();
      await tester.pumpAndSettle();

      expect(find.text('数学'), findsOneWidget);
      expect(find.text('2026-03-06 任务板'), findsOneWidget);
      expect(repository.fetchRequests.length, 2);
    });

    testWidgets('switches date and refreshes manually', (tester) async {
      final repository = _FakeTaskBoardRepository(
        onFetch: (request) async => _boardWithTasks(date: request.date),
      );

      await tester.pumpWidget(
        StudyClawPadApp(
          autoLoad: false,
          initialDate: '2026-03-06',
          repository: repository,
        ),
      );

      expect(find.text('准备同步任务板'), findsOneWidget);

      await tester.tap(find.byTooltip('下一天'));
      await tester.pump();
      await tester.pumpAndSettle();

      expect(
        repository.fetchRequests.map((request) => request.date).toList(),
        <String>['2026-03-07'],
      );
      expect(find.text('2026-03-07 任务板'), findsOneWidget);

      await tester.tap(find.byTooltip('手动刷新'));
      await tester.pump();
      await tester.pumpAndSettle();

      expect(
        repository.fetchRequests.map((request) => request.date).toList(),
        <String>['2026-03-07', '2026-03-07'],
      );
      expect(find.text('任务板已手动刷新'), findsOneWidget);
    });
  });
}

class _FakeTaskBoardRepository implements TaskBoardRepository {
  _FakeTaskBoardRepository({
    required this.onFetch,
    this.onUpdateSingleTask,
    this.onUpdateTaskGroup,
    this.onUpdateAllTasks,
    TaskBoard? fallbackBoard,
  }) : _lastBoard = fallbackBoard ?? _boardWithTasks();

  final Future<TaskBoard> Function(TaskBoardRequest request) onFetch;
  final Future<TaskBoard> Function(
    TaskBoardRequest request,
    int taskId,
    bool completed,
  )? onUpdateSingleTask;
  final Future<TaskBoard> Function(
    TaskBoardRequest request,
    String subject,
    String? groupTitle,
    bool completed,
  )? onUpdateTaskGroup;
  final Future<TaskBoard> Function(
    TaskBoardRequest request,
    bool completed,
  )? onUpdateAllTasks;

  final List<TaskBoardRequest> fetchRequests = <TaskBoardRequest>[];
  final List<_SingleTaskUpdateCall> singleTaskUpdates =
      <_SingleTaskUpdateCall>[];
  final List<_GroupUpdateCall> groupUpdates = <_GroupUpdateCall>[];
  final List<bool> bulkUpdates = <bool>[];

  TaskBoard _lastBoard;

  @override
  Future<TaskBoard> fetchBoard(TaskBoardRequest request) async {
    fetchRequests.add(request);
    final board = await onFetch(request);
    _lastBoard = board;
    return board;
  }

  @override
  Future<TaskBoard> updateAllTasks(
    TaskBoardRequest request, {
    required bool completed,
  }) async {
    bulkUpdates.add(completed);
    final board = await (onUpdateAllTasks?.call(request, completed) ??
        Future<TaskBoard>.value(_lastBoard));
    _lastBoard = board;
    return board;
  }

  @override
  Future<TaskBoard> updateSingleTask(
    TaskBoardRequest request, {
    required int taskId,
    required bool completed,
  }) async {
    singleTaskUpdates.add(
      _SingleTaskUpdateCall(taskId: taskId, completed: completed),
    );
    final board = await (onUpdateSingleTask?.call(request, taskId, completed) ??
        Future<TaskBoard>.value(_lastBoard));
    _lastBoard = board;
    return board;
  }

  @override
  Future<TaskBoard> updateTaskGroup(
    TaskBoardRequest request, {
    required String subject,
    String? groupTitle,
    required bool completed,
  }) async {
    groupUpdates.add(
      _GroupUpdateCall(
        subject: subject,
        groupTitle: groupTitle,
        completed: completed,
      ),
    );
    final board = await (onUpdateTaskGroup?.call(
          request,
          subject,
          groupTitle,
          completed,
        ) ??
        Future<TaskBoard>.value(_lastBoard));
    _lastBoard = board;
    return board;
  }
}

class _SingleTaskUpdateCall {
  const _SingleTaskUpdateCall({
    required this.taskId,
    required this.completed,
  });

  final int taskId;
  final bool completed;
}

class _GroupUpdateCall {
  const _GroupUpdateCall({
    required this.subject,
    required this.groupTitle,
    required this.completed,
  });

  final String subject;
  final String? groupTitle;
  final bool completed;
}

class _TaskSeed {
  const _TaskSeed({
    required this.taskId,
    required this.subject,
    required this.groupTitle,
    required this.content,
    this.completed = false,
  });

  final int taskId;
  final String subject;
  final String groupTitle;
  final String content;
  final bool completed;
}

Future<void> _pumpLoadedBoard(
  WidgetTester tester, {
  required _FakeTaskBoardRepository repository,
  String initialDate = '2026-03-06',
}) async {
  await tester.pumpWidget(
    StudyClawPadApp(
      autoLoad: true,
      initialDate: initialDate,
      repository: repository,
    ),
  );

  await tester.pump();
  await tester.pumpAndSettle();
}

void _setLargeViewport(WidgetTester tester) {
  tester.view.physicalSize = const Size(1200, 2000);
  tester.view.devicePixelRatio = 1.0;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
}

TaskBoard _buildBoard({
  String date = '2026-03-06',
  required List<_TaskSeed> tasks,
}) {
  final taskItems = tasks
      .map(
        (task) => TaskItem(
          taskId: task.taskId,
          subject: task.subject,
          groupTitle: task.groupTitle,
          content: task.content,
          completed: task.completed,
          status: task.completed ? 'completed' : 'pending',
        ),
      )
      .toList();

  final groupBuckets = <String, List<_TaskSeed>>{};
  final homeworkBuckets = <String, List<_TaskSeed>>{};

  for (final task in tasks) {
    groupBuckets.putIfAbsent(task.subject, () => <_TaskSeed>[]).add(task);
    final homeworkKey = '${task.subject}::${task.groupTitle}';
    homeworkBuckets.putIfAbsent(homeworkKey, () => <_TaskSeed>[]).add(task);
  }

  final groups = groupBuckets.entries.map((entry) {
    final completedCount = entry.value.where((task) => task.completed).length;
    final total = entry.value.length;
    return TaskGroup(
      subject: entry.key,
      total: total,
      completed: completedCount,
      pending: total - completedCount,
      status: _statusForCounts(total, completedCount),
    );
  }).toList();

  final homeworkGroups = homeworkBuckets.entries.map((entry) {
    final bucket = entry.value;
    final firstTask = bucket.first;
    final completedCount = bucket.where((task) => task.completed).length;
    final total = bucket.length;
    return HomeworkGroup(
      subject: firstTask.subject,
      groupTitle: firstTask.groupTitle,
      total: total,
      completed: completedCount,
      pending: total - completedCount,
      status: _statusForCounts(total, completedCount),
    );
  }).toList();

  final completedCount = tasks.where((task) => task.completed).length;

  return TaskBoard(
    date: date,
    message: '同步完成',
    tasks: taskItems,
    groups: groups,
    homeworkGroups: homeworkGroups,
    summary: BoardSummary(
      total: tasks.length,
      completed: completedCount,
      pending: tasks.length - completedCount,
      status: _statusForCounts(tasks.length, completedCount),
    ),
  );
}

String _statusForCounts(int total, int completed) {
  if (total == 0) {
    return 'empty';
  }
  if (completed == 0) {
    return 'pending';
  }
  if (completed == total) {
    return 'completed';
  }
  return 'partial';
}

TaskBoard _boardWithTasks({String date = '2026-03-06'}) {
  return _buildBoard(
    date: date,
    tasks: const <_TaskSeed>[
      _TaskSeed(
        taskId: 1,
        subject: '数学',
        groupTitle: '口算练习',
        content: '完成第 1 页',
      ),
    ],
  );
}

TaskBoard _emptyBoard({String date = '2026-03-06'}) {
  return TaskBoard(
    date: date,
    message: '当前日期没有任务',
    tasks: const <TaskItem>[],
    groups: const <TaskGroup>[],
    homeworkGroups: const <HomeworkGroup>[],
    summary: const BoardSummary(
      total: 0,
      completed: 0,
      pending: 0,
      status: 'empty',
    ),
  );
}
