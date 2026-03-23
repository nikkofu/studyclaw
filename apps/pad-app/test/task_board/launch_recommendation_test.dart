import 'package:flutter_test/flutter_test.dart';
import 'package:pad_app/task_board/controller.dart';
import 'package:pad_app/task_board/models.dart';
import 'package:pad_app/task_board/repository.dart';

class _NoopTaskBoardRepository implements TaskBoardRepository {
  @override
  noSuchMethod(Invocation invocation) =>
      throw UnimplementedError('Not needed for resolveLaunchTask tests');
}

void main() {
  group('TaskBoardController.resolveLaunchTask', () {
    late TaskBoardController controller;

    setUp(() {
      controller = TaskBoardController(repository: _NoopTaskBoardRepository());
    });

    test('uses launch recommendation item first', () {
      final board = TaskBoard(
        date: '2026-03-12',
        message: null,
        tasks: const [
          TaskItem(
            taskId: 100,
            subject: '语文',
            groupTitle: '背诵',
            content: '背诵第一段',
            completed: false,
            status: 'pending',
          ),
          TaskItem(
            taskId: 200,
            subject: '数学',
            groupTitle: '订正',
            content: '订正错题',
            completed: false,
            status: 'pending',
          ),
        ],
        groups: const [],
        homeworkGroups: const [],
        summary: const BoardSummary(
          total: 2,
          completed: 0,
          pending: 2,
          status: 'pending',
        ),
        launchRecommendation: const LaunchRecommendation(
          reasonCode: 'first_unfinished',
          groupId: '数学\x00订正',
          itemId: 200,
          whyRecommended: null,
        ),
      );

      final resolved = controller.resolveLaunchTask(board);
      expect(resolved, isNotNull);
      expect(resolved!.taskId, 200);
    });

    test('falls back to first unfinished when recommendation invalid', () {
      final board = TaskBoard(
        date: '2026-03-12',
        message: null,
        tasks: const [
          TaskItem(
            taskId: 101,
            subject: '英语',
            groupTitle: '默写',
            content: '默写单词',
            completed: false,
            status: 'pending',
          ),
          TaskItem(
            taskId: 102,
            subject: '数学',
            groupTitle: '校本',
            content: '完成校本',
            completed: false,
            status: 'pending',
          ),
        ],
        groups: const [],
        homeworkGroups: const [],
        summary: const BoardSummary(
          total: 2,
          completed: 0,
          pending: 2,
          status: 'pending',
        ),
        launchRecommendation: const LaunchRecommendation(
          reasonCode: 'first_unfinished',
          groupId: '不存在\x00不存在',
          itemId: 9999,
          whyRecommended: null,
        ),
      );

      final resolved = controller.resolveLaunchTask(board);
      expect(resolved, isNotNull);
      expect(resolved!.taskId, 101);
    });

    test('returns null when all tasks completed', () {
      final board = TaskBoard(
        date: '2026-03-12',
        message: null,
        tasks: const [
          TaskItem(
            taskId: 1,
            subject: '语文',
            groupTitle: '阅读',
            content: '阅读课文',
            completed: true,
            status: 'completed',
          ),
        ],
        groups: const [],
        homeworkGroups: const [],
        summary: const BoardSummary(
          total: 1,
          completed: 1,
          pending: 0,
          status: 'completed',
        ),
        launchRecommendation: const LaunchRecommendation(
          reasonCode: 'first_unfinished',
          groupId: '语文\x00阅读',
          itemId: 1,
          whyRecommended: null,
        ),
      );

      final resolved = controller.resolveLaunchTask(board);
      expect(resolved, isNull);
    });
  });
}
