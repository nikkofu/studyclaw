import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:integration_test/integration_test.dart';
import 'package:pad_app/app.dart';
import 'package:pad_app/task_board/controller.dart';
import 'package:pad_app/task_board/models.dart';
import 'package:pad_app/task_board/repository.dart';

const String liveApiBaseUrl = String.fromEnvironment(
  'API_BASE_URL',
  defaultValue: 'http://127.0.0.1:18081',
);

final int _liveRunId = DateTime.now().millisecondsSinceEpoch.remainder(100000);

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  group('real backend chrome smoke', () {
    testWidgets('loads task board from the live backend', (tester) async {
      _setLargeViewport(tester);
      final scenario = _Scenario(
        familyId: _familyIdFor(1),
        userId: 1,
        date: '2026-04-01',
      );
      final content = _uniqueContent('完成第 1 页');
      await _seedTasks(
        scenario,
        <_TaskSeed>[
          _TaskSeed(
            subject: '数学',
            groupTitle: '口算练习',
            content: content,
          ),
        ],
      );

      await _pumpLiveApp(tester, scenario);

      await _pumpUntilFound(tester, find.text('数学'));
      final title = tester.widget<Text>(
        find.byKey(const Key('today_hero_title')),
      );
      expect(title.data, '2026-04-01 任务板');
      expect(find.text(content), findsOneWidget);
    });

    testWidgets('single task sync updates the live backend', (tester) async {
      _setLargeViewport(tester);
      final scenario = _Scenario(
        familyId: _familyIdFor(2),
        userId: 1,
        date: '2026-04-02',
      );
      final content = _uniqueContent('完成第 1 页');
      await _seedTasks(
        scenario,
        <_TaskSeed>[
          _TaskSeed(
            subject: '数学',
            groupTitle: '口算练习',
            content: content,
          ),
        ],
      );

      await _pumpLiveApp(tester, scenario);
      await _pumpUntilFound(tester, find.byType(CheckboxListTile));

      await tester.tap(find.byType(CheckboxListTile).first);
      await tester.pump();
      await _pumpUntilFound(tester, find.text('已同步单个任务完成状态'));

      final board = await _fetchBoard(scenario);
      expect(board.summary.completed, 1);
      expect(board.tasks.first.completed, isTrue);
    });

    testWidgets('group sync updates the live backend', (tester) async {
      _setLargeViewport(tester);
      final scenario = _Scenario(
        familyId: _familyIdFor(3),
        userId: 1,
        date: '2026-04-03',
      );
      await _seedTasks(
        scenario,
        <_TaskSeed>[
          _TaskSeed(
            subject: '英语',
            groupTitle: '预习M1U2',
            content: _uniqueContent('书本上标注音标'),
          ),
          _TaskSeed(
            subject: '英语',
            groupTitle: '预习M1U2',
            content: _uniqueContent('抄写单词'),
          ),
          _TaskSeed(
            subject: '英语',
            groupTitle: '背默',
            content: _uniqueContent('背默课文'),
          ),
        ],
      );

      await _pumpLiveApp(tester, scenario);
      await _pumpUntilFound(tester, find.text('分组完成'));

      await tester.tap(find.text('分组完成').first);
      await tester.pump();
      await _pumpUntilFound(tester, find.text('已将 预习M1U2 分组标记为完成'));

      final board = await _fetchBoard(scenario);
      final previewTasks =
          board.tasks.where((task) => task.groupTitle == '预习M1U2').toList();
      expect(previewTasks.every((task) => task.completed), isTrue);
      expect(board.summary.completed, 2);
      expect(board.summary.pending, 1);
    });

    testWidgets('complete all updates the live backend', (tester) async {
      _setLargeViewport(tester);
      final scenario = _Scenario(
        familyId: _familyIdFor(4),
        userId: 1,
        date: '2026-04-04',
      );
      await _seedTasks(
        scenario,
        <_TaskSeed>[
          _TaskSeed(
            subject: '数学',
            groupTitle: '口算练习',
            content: _uniqueContent('完成第 1 页'),
          ),
          _TaskSeed(
            subject: '英语',
            groupTitle: '背默',
            content: _uniqueContent('背默单词'),
          ),
        ],
      );

      await _pumpLiveApp(tester, scenario);
      await _pumpUntilFound(
        tester,
        find.byKey(const Key('today_hero_complete_all_button')),
      );

      await tester.tap(find.byKey(const Key('today_hero_complete_all_button')));
      await tester.pump();
      await _pumpUntilFound(tester, find.text('已将全部任务同步为完成'));

      final board = await _fetchBoard(scenario);
      expect(board.summary.completed, 2);
      expect(board.summary.pending, 0);
      expect(board.tasks.every((task) => task.completed), isTrue);
    });

    testWidgets('404 from the live backend maps to a clear pad message', (
      tester,
    ) async {
      final scenario = _Scenario(
        familyId: _familyIdFor(5),
        userId: 1,
        date: '2026-04-05',
      );
      await _seedTasks(
        scenario,
        <_TaskSeed>[
          _TaskSeed(
            subject: '数学',
            groupTitle: '口算练习',
            content: _uniqueContent('完成第 1 页'),
          ),
        ],
      );

      final controller = TaskBoardController(
        repository: const RemoteTaskBoardRepository(),
      );
      final request = scenario.toRequest();

      await controller.loadBoard(request, showLoadingState: true);
      await controller.updateSingleTask(
        request,
        const TaskItem(
          taskId: 999,
          subject: '数学',
          groupTitle: '口算练习',
          content: '不存在的任务',
          completed: false,
          status: 'pending',
        ),
        true,
      );

      expect(controller.state.errorMessage, '任务 #999 不存在，可能已被删除或日期已变更。');
      expect(controller.state.noticeMessage, isNull);
      controller.dispose();
    });

    testWidgets('409 from the live backend shows an info prompt',
        (tester) async {
      _setLargeViewport(tester);
      final scenario = _Scenario(
        familyId: _familyIdFor(6),
        userId: 1,
        date: '2026-04-06',
      );
      await _seedTasks(
        scenario,
        <_TaskSeed>[
          _TaskSeed(
            subject: '语文',
            groupTitle: '背作文',
            content: _uniqueContent('背作文'),
          ),
          _TaskSeed(
            subject: '语文',
            groupTitle: '练习卷',
            content: _uniqueContent('完成练习卷'),
          ),
        ],
      );

      await _pumpLiveApp(tester, scenario);
      await _pumpUntilFound(
        tester,
        find.byKey(const Key('today_hero_complete_all_button')),
      );

      await _patchAllTasks(scenario, completed: true);
      await tester.tap(find.byKey(const Key('today_hero_complete_all_button')));
      await tester.pump();
      await _pumpUntilFound(tester, find.text('全部任务已经是已完成状态，无需重复同步。'));

      expect(find.textContaining('请求失败'), findsNothing);
    });
  });
}

class _Scenario {
  const _Scenario({
    required this.familyId,
    required this.userId,
    required this.date,
  });

  final int familyId;
  final int userId;
  final String date;

  TaskBoardRequest toRequest() {
    return TaskBoardRequest(
      apiBaseUrl: liveApiBaseUrl,
      familyId: familyId,
      userId: userId,
      date: date,
    );
  }
}

class _TaskSeed {
  const _TaskSeed({
    required this.subject,
    required this.groupTitle,
    required this.content,
  });

  final String subject;
  final String groupTitle;
  final String content;
}

int _familyIdFor(int offset) {
  return 430000 + (_liveRunId * 10) + offset;
}

String _uniqueContent(String base) {
  return '$base [$_liveRunId]';
}

Future<void> _pumpLiveApp(WidgetTester tester, _Scenario scenario) async {
  await tester.pumpWidget(
    StudyClawPadApp(
      autoLoad: true,
      initialApiBaseUrl: liveApiBaseUrl,
      initialFamilyId: scenario.familyId,
      initialUserId: scenario.userId,
      initialDate: scenario.date,
    ),
  );

  await tester.pump();
}

Future<void> _seedTasks(_Scenario scenario, List<_TaskSeed> tasks) async {
  for (final task in tasks) {
    final response = await http.post(
      _buildUri('/api/v1/tasks'),
      headers: const {'content-type': 'application/json'},
      body: jsonEncode({
        'family_id': scenario.familyId,
        'assignee_id': scenario.userId,
        'subject': task.subject,
        'group_title': task.groupTitle,
        'content': task.content,
        'assigned_date': scenario.date,
      }),
    );
    expect(
      response.statusCode,
      201,
      reason: 'seed task failed: ${response.statusCode} ${response.body}',
    );
  }
}

Future<void> _patchAllTasks(
  _Scenario scenario, {
  required bool completed,
}) async {
  final response = await http.patch(
    _buildUri('/api/v1/tasks/status/all'),
    headers: const {'content-type': 'application/json'},
    body: jsonEncode({
      'family_id': scenario.familyId,
      'assignee_id': scenario.userId,
      'completed': completed,
      'assigned_date': scenario.date,
    }),
  );

  expect(
    response.statusCode,
    200,
    reason: 'patch all failed: ${response.statusCode} ${response.body}',
  );
}

Future<TaskBoard> _fetchBoard(_Scenario scenario) async {
  final response = await http.get(
    _buildUri(
      '/api/v1/tasks',
      query: <String, String>{
        'family_id': '${scenario.familyId}',
        'user_id': '${scenario.userId}',
        'date': scenario.date,
      },
    ),
  );

  expect(
    response.statusCode,
    200,
    reason: 'fetch board failed: ${response.statusCode} ${response.body}',
  );

  return TaskBoard.fromJson(
    jsonDecode(response.body) as Map<String, dynamic>,
  );
}

Uri _buildUri(String path, {Map<String, String>? query}) {
  return Uri.parse('$liveApiBaseUrl$path').replace(queryParameters: query);
}

Future<void> _pumpUntilFound(
  WidgetTester tester,
  Finder finder, {
  Duration timeout = const Duration(seconds: 10),
}) async {
  final deadline = DateTime.now().add(timeout);
  while (DateTime.now().isBefore(deadline)) {
    await tester.pump(const Duration(milliseconds: 100));
    if (finder.evaluate().isNotEmpty) {
      return;
    }
  }

  fail('Timed out waiting for finder: $finder');
}

void _setLargeViewport(WidgetTester tester) {
  tester.view.physicalSize = const Size(1400, 2200);
  tester.view.devicePixelRatio = 1.0;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
}
