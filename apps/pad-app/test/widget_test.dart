import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:pad_app/app.dart';
import 'package:pad_app/task_board/daily_stats.dart';
import 'package:pad_app/task_board/models.dart';
import 'package:pad_app/task_board/repository.dart';
import 'package:pad_app/task_board/weekly_stats.dart';
import 'package:pad_app/word_playback/controller.dart';
import 'package:pad_app/word_playback/models.dart';
import 'package:pad_app/word_playback/speaker_contract.dart';

void main() {
  group('PadTaskBoardPage Widget Tests', () {
    testWidgets('shows loading state', (tester) async {
      final repository =
          _FakeTaskBoardRepository(onFetch: (_) async => _boardWithTasks());
      await tester
          .pumpWidget(StudyClawPadApp(autoLoad: true, repository: repository));
      expect(find.text('同步中'), findsOneWidget);
    });

    testWidgets('shows a kid-friendly grading journey after photo submission',
        (tester) async {
      const startSession = DictationSession(
        sessionId: 'session_trace_001',
        wordListId: 'word_list_trace_001',
        status: 'active',
        currentIndex: 0,
        totalItems: 3,
        playedCount: 0,
        completedItems: 0,
        currentItem: WordItem(index: 1, text: 'apple', meaning: '苹果'),
        gradingStatus: 'idle',
      );
      const queuedSession = DictationSession(
        sessionId: 'session_trace_001',
        wordListId: 'word_list_trace_001',
        status: 'completed',
        currentIndex: 2,
        totalItems: 3,
        playedCount: 3,
        completedItems: 3,
        gradingStatus: 'pending',
        gradingRequestedAt: '2026-03-12T08:10:00Z',
        debugContext: DictationDebugContext(
          photoSha1: 'abcdef123456',
          photoBytes: 24,
          language: 'english',
          mode: 'word',
          workerStage: 'queued',
        ),
      );
      const completedSession = DictationSession(
        sessionId: 'session_trace_001',
        wordListId: 'word_list_trace_001',
        status: 'completed',
        currentIndex: 2,
        totalItems: 3,
        playedCount: 3,
        completedItems: 3,
        gradingStatus: 'completed',
        gradingRequestedAt: '2026-03-12T08:10:00Z',
        gradingCompletedAt: '2026-03-12T08:10:12Z',
        debugContext: DictationDebugContext(
          photoSha1: 'abcdef123456',
          photoBytes: 24,
          language: 'english',
          mode: 'word',
          workerStage: 'completed',
        ),
        gradingResult: DictationGradingResult(
          gradingId: 'grading_trace_001',
          status: 'needs_correction',
          score: 92,
          gradedItems: [
            DictationGradedItem(
              index: 2,
              expected: 'library',
              actual: 'libary',
              isCorrect: false,
              needsCorrection: true,
              comment: '少了 r',
            ),
          ],
          aiFeedback: '建议把 library 的字母顺序再看一遍。',
          createdAt: '2026-03-12T08:10:12Z',
        ),
      );

      final repository = _FakeTaskBoardRepository(
        onFetch: (_) async => _boardWithTasks(),
        onStartDictation: (_) async => startSession,
        onGradeDictationSession: (_, __, ___, ____, _____) async =>
            queuedSession,
        onGetDictationSession: (_, __) async => completedSession,
      );
      final controller = WordPlaybackController(
        speaker: const _FakeWordSpeaker(),
        repository: repository,
      );
      final previewBytes = base64Decode(
        'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO5WnYsAAAAASUVORK5CYII=',
      );
      const request = TaskBoardRequest(
        apiBaseUrl: 'http://localhost:8080',
        familyId: 306,
        userId: 1,
        date: '2026-03-12',
      );

      await tester.pumpWidget(StudyClawPadApp(
        autoLoad: false,
        repository: repository,
        wordPlaybackController: controller,
      ));
      await tester.tap(find.text('听写练词'));
      await tester.pumpAndSettle();

      await controller.startDictation(request);
      await tester.pump();
      await controller.submitPhotoForGrading(
        request.apiBaseUrl,
        'ZmFrZS1pbWFnZQ==',
        previewBytes: previewBytes,
        submittedAt: DateTime(2026, 3, 12, 8, 10),
      );
      await tester.pump();

      expect(find.text('最近一次交卷'), findsOneWidget);
      expect(find.textContaining('先看看照片是否清楚'), findsOneWidget);
      expect(find.textContaining('提交 08:10'), findsOneWidget);
      expect(find.text('交卷进度'), findsOneWidget);
      expect(find.text('拍照完成'), findsOneWidget);
      expect(find.text('云端接收'), findsOneWidget);
      expect(find.text('AI 比对'), findsOneWidget);
      expect(find.text('结果回到平板'), findsOneWidget);
      expect(find.textContaining('会话 session_trace_001'), findsWidgets);
      expect(find.textContaining('图片 abcdef12'), findsWidgets);
      expect(find.textContaining('云端排队'), findsWidgets);

      await tester.pump(const Duration(seconds: 2));
      await tester.pumpAndSettle();

      expect(find.text('批改完成 · 92 分'), findsOneWidget);
      expect(find.textContaining('结果同步'), findsWidgets);
      expect(find.text('错题 1 / 1'), findsOneWidget);
      expect(find.text('建议把 library 的字母顺序再看一遍。'), findsOneWidget);
    });
  });
}

class _FakeTaskBoardRepository implements TaskBoardRepository {
  _FakeTaskBoardRepository(
      {required this.onFetch,
      this.onStartDictation,
      this.onGetDictationSession,
      this.onGradeDictationSession,
      TaskBoard? fallbackBoard})
      : _lastBoard = fallbackBoard ?? _boardWithTasks();

  final Future<TaskBoard> Function(TaskBoardRequest request) onFetch;
  final Future<DictationSession> Function(TaskBoardRequest request)?
      onStartDictation;
  final Future<DictationSession> Function(String sessionId, String apiBaseUrl)?
      onGetDictationSession;
  final Future<DictationSession> Function(
      String sessionId,
      String apiBaseUrl,
      String photoBase64,
      String language,
      String mode)? onGradeDictationSession;

  final List<_SingleTaskUpdateCall> singleTaskUpdates = [];
  final TaskBoard _lastBoard;

  @override
  Future<TaskBoard> fetchBoard(TaskBoardRequest request) async =>
      await onFetch(request);
  @override
  Future<DailyStats> fetchDailyStats(TaskBoardRequest request) async =>
      Future.value(const DailyStats(
          period: '',
          startDate: '',
          endDate: '',
          encouragement: '',
          totals: StatsTotals(
              totalTasks: 0,
              completedTasks: 0,
              pendingTasks: 0,
              completionRate: 0,
              autoPoints: 0,
              manualPoints: 0,
              totalPointsDelta: 0,
              pointsBalance: 0,
              wordItems: 0,
              completedWordItems: 0,
              dictationSessions: 0)));
  @override
  Future<WeeklyStats> fetchWeeklyStats(TaskBoardRequest request) async =>
      Future.value(const WeeklyStats(message: '', days: []));
  @override
  Future<Map<String, dynamic>> fetchMonthlyStats(
          TaskBoardRequest request) async =>
      Future.value(<String, dynamic>{});
  @override
  Future<TaskBoard> updateSingleTask(TaskBoardRequest request,
      {required int taskId, required bool completed}) async {
    singleTaskUpdates
        .add(_SingleTaskUpdateCall(taskId: taskId, completed: completed));
    return _lastBoard;
  }

  @override
  Future<TaskBoard> updateTaskGroup(TaskBoardRequest request,
          {required String subject,
          String? groupTitle,
          required bool completed}) async =>
      _lastBoard;
  @override
  Future<TaskBoard> updateAllTasks(TaskBoardRequest request,
          {required bool completed}) async =>
      _lastBoard;
  @override
  Future<WordList> fetchWordList(TaskBoardRequest request) async =>
      Future.value(const WordList(
          wordListId: '0',
          familyId: 0,
          childId: 0,
          assignedDate: '',
          title: '',
          language: WordPlaybackLanguage.english,
          items: [],
          totalItems: 0));
  @override
  Future<DictationSession> startDictationSession(
          TaskBoardRequest request) async =>
      await (onStartDictation?.call(request) ?? Future.value(_emptySession));
  @override
  Future<DictationSession> nextDictationSession(
          String sessionId, String apiBaseUrl) async =>
      Future.value(_emptySession);
  @override
  Future<DictationSession> previousDictationSession(
          String sessionId, String apiBaseUrl) async =>
      Future.value(_emptySession);
  @override
  Future<DictationSession> getDictationSession(
          String sessionId, String apiBaseUrl) async =>
      await (onGetDictationSession?.call(sessionId, apiBaseUrl) ??
          Future.value(_emptySession));
  @override
  Future<DictationSession> replayDictationSession(
          String sessionId, String apiBaseUrl) async =>
      Future.value(_emptySession);
  @override
  Future<DictationSession> gradeDictationSession(
          {required String sessionId,
          required String apiBaseUrl,
          required String photoBase64,
          required String language,
          required String mode}) async =>
      await (onGradeDictationSession?.call(
              sessionId, apiBaseUrl, photoBase64, language, mode) ??
          Future.value(_emptySession));
}

class _SingleTaskUpdateCall {
  const _SingleTaskUpdateCall({required this.taskId, required this.completed});
  final int taskId;
  final bool completed;
}

TaskBoard _buildBoard({required List<_TaskSeed> tasks}) {
  return TaskBoard(
      date: '2026-03-06',
      message: 'OK',
      tasks: tasks
          .map((t) => TaskItem(
              taskId: t.taskId,
              subject: t.subject,
              groupTitle: t.groupTitle,
              content: t.content,
              completed: t.completed,
              status: t.completed ? 'completed' : 'pending'))
          .toList(),
      groups: [],
      homeworkGroups: [],
      summary: BoardSummary(
          total: tasks.length,
          completed: tasks.where((t) => t.completed).length,
          pending: tasks.where((t) => !t.completed).length,
          status: 'partial'));
}

TaskBoard _boardWithTasks() => _buildBoard(tasks: const [
      _TaskSeed(
          taskId: 1,
          subject: '数学',
          groupTitle: 'G1',
          content: 'T1',
          completed: false)
    ]);

const DictationSession _emptySession = DictationSession(
    sessionId: '0',
    wordListId: '0',
    currentIndex: 0,
    status: '',
    totalItems: 0,
    playedCount: 0,
    completedItems: 0,
    gradingStatus: 'idle');

class _TaskSeed {
  const _TaskSeed(
      {required this.taskId,
      required this.subject,
      required this.groupTitle,
      required this.content,
      required this.completed});
  final int taskId;
  final String subject;
  final String groupTitle;
  final String content;
  final bool completed;
}

class _FakeWordSpeaker implements WordSpeaker {
  const _FakeWordSpeaker();

  @override
  bool get supportsPlayback => true;

  @override
  Future<void> speak(String text,
      {required WordPlaybackLanguage language}) async {}

  @override
  Future<void> stop() async {}
}
