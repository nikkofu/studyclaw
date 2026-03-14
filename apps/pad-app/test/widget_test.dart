import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pad_app/app.dart';
import 'package:pad_app/task_board/api_client.dart';
import 'package:pad_app/task_board/daily_stats.dart';
import 'package:pad_app/task_board/models.dart';
import 'package:pad_app/task_board/recitation_analysis.dart';
import 'package:pad_app/task_board/repository.dart';
import 'package:pad_app/task_board/weekly_stats.dart';
import 'package:pad_app/voice_commands/models.dart';
import 'package:pad_app/voice_commands/speech_recognizer_contract.dart';
import 'package:pad_app/word_playback/controller.dart';
import 'package:pad_app/word_playback/models.dart';
import 'package:pad_app/word_playback/speaker_contract.dart';

void main() {
  group('PadTaskBoardPage Widget Tests', () {
    testWidgets('renders page shell while loading', (tester) async {
      final boardCompleter = Completer<TaskBoard>();
      final repository =
          _FakeTaskBoardRepository(onFetch: (_) async => boardCompleter.future);
      await tester
          .pumpWidget(StudyClawPadApp(autoLoad: true, repository: repository));
      await tester.pump();
      expect(find.text('挑战舞台'), findsOneWidget);
      expect(find.text('孩子学习语音工作台'), findsOneWidget);
      boardCompleter.complete(_boardWithTasks());
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
        speaker: _FakeWordSpeaker(),
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
      expect(controller.state.lastSubmission, isNotNull);
      expect(controller.state.session?.gradingStatus, 'pending');
      expect(controller.state.session?.debugContext?.photoSha1, 'abcdef123456');

      await tester.pump(const Duration(seconds: 2));
      await tester.pumpAndSettle();

      expect(controller.state.session?.gradingStatus, 'completed');
      expect(controller.state.session?.gradingResult?.score, 92);
      expect(
        controller.state.session?.gradingResult?.aiFeedback,
        '建议把 library 的字母顺序再看一遍。',
      );
    });

    testWidgets('applies voice command to advance dictation', (tester) async {
      const startSession = DictationSession(
        sessionId: 'session_voice_001',
        wordListId: 'word_list_voice_001',
        status: 'active',
        currentIndex: 0,
        totalItems: 2,
        playedCount: 0,
        completedItems: 0,
        currentItem: WordItem(index: 1, text: 'apple', meaning: '苹果'),
        gradingStatus: 'idle',
      );
      const nextSession = DictationSession(
        sessionId: 'session_voice_001',
        wordListId: 'word_list_voice_001',
        status: 'active',
        currentIndex: 1,
        totalItems: 2,
        playedCount: 1,
        completedItems: 1,
        currentItem: WordItem(index: 2, text: 'library', meaning: '图书馆'),
        gradingStatus: 'idle',
      );
      final repository = _FakeTaskBoardRepository(
        onFetch: (_) async => _boardWithTasks(),
        onStartDictation: (_) async => startSession,
        onNextDictationSession: (_, __) async => nextSession,
        onResolveVoiceCommand: (_, __, ___) async =>
            const VoiceCommandResolution(
          action: 'dictation_next',
          reason: '听写场景里“好了”表示进入下一词',
          parserMode: 'rule_fallback',
          confidence: 0.9,
          normalizedTranscript: '好了',
          surface: VoiceCommandSurface.dictation,
          target: VoiceCommandTarget(sessionId: 'session_voice_001'),
        ),
      );
      final controller = WordPlaybackController(
        speaker: _FakeWordSpeaker(),
        repository: repository,
      );
      const request = TaskBoardRequest(
        apiBaseUrl: 'http://localhost:8080',
        familyId: 306,
        userId: 1,
        date: '2026-03-12',
      );

      await tester.pumpWidget(
        StudyClawPadApp(
          autoLoad: false,
          repository: repository,
          wordPlaybackController: controller,
          speechRecognizer: _FakeSpeechRecognizer(
            transcript: const SpeechTranscript(
              transcript: '好了',
              locale: 'en-US',
            ),
          ),
        ),
      );
      await tester.tap(find.text('听写练词'));
      await tester.pumpAndSettle();

      await controller.startDictation(request);
      await tester.pump();

      await tester.tap(find.byKey(const Key('voice-assistant-trigger')));
      await tester.pump();
      expect(find.text('结束说话'), findsOneWidget);
      await tester.tap(find.byKey(const Key('voice-assistant-trigger')));
      await tester.pumpAndSettle();

      expect(repository.nextDictationCalls.length, 1);
      expect(
        find.textContaining('已根据“好了”切到下一词'),
        findsOneWidget,
      );
    });

    testWidgets('shows a friendly waiting state when the word list is missing',
        (tester) async {
      final repository = _FakeTaskBoardRepository(
        onFetch: (_) async => _boardWithTasks(),
        onFetchWordList: (_) async => throw TaskApiException(
          message: 'word list not found',
          errorCode: 'word_list_not_found',
          details: const <String, dynamic>{'date': '2026-03-13'},
          uri: Uri.parse('http://localhost:8080/api/v1/word-lists'),
          statusCode: 404,
        ),
      );
      final controller = WordPlaybackController(
        speaker: _FakeWordSpeaker(),
        repository: repository,
      );
      const request = TaskBoardRequest(
        apiBaseUrl: 'http://localhost:8080',
        familyId: 306,
        userId: 1,
        date: '2026-03-13',
      );

      await tester.pumpWidget(
        StudyClawPadApp(
          autoLoad: false,
          repository: repository,
          wordPlaybackController: controller,
        ),
      );
      await tester.tap(find.text('听写练词'));
      await tester.pumpAndSettle();

      await controller.syncWordList(request);
      await tester.pumpAndSettle();

      expect(controller.state.waitingForParentWordList, isTrue);
      expect(controller.state.errorMessage, isNull);
      expect(
        controller.state.noticeMessage,
        contains('默写词单还没准备好'),
      );
      expect(
        controller.state.noticeMessage,
        isNot(contains('TaskApiException')),
      );
    });

    testWidgets('applies voice command to complete a subject group',
        (tester) async {
      final board = TaskBoard(
        date: '2026-03-12',
        message: 'OK',
        tasks: const [
          TaskItem(
            taskId: 1,
            subject: '数学',
            groupTitle: '订正',
            content: '订正第 3 课错题',
            completed: false,
            status: 'pending',
          ),
          TaskItem(
            taskId: 2,
            subject: '数学',
            groupTitle: '一课一练',
            content: '完成一课一练第 5 页',
            completed: false,
            status: 'pending',
          ),
        ],
        groups: const [
          TaskGroup(
            subject: '数学',
            total: 2,
            completed: 0,
            pending: 2,
            status: 'pending',
          ),
        ],
        homeworkGroups: const [
          HomeworkGroup(
            subject: '数学',
            groupTitle: '订正',
            total: 1,
            completed: 0,
            pending: 1,
            status: 'pending',
          ),
          HomeworkGroup(
            subject: '数学',
            groupTitle: '一课一练',
            total: 1,
            completed: 0,
            pending: 1,
            status: 'pending',
          ),
        ],
        summary: const BoardSummary(
          total: 2,
          completed: 0,
          pending: 2,
          status: 'pending',
        ),
      );
      final repository = _FakeTaskBoardRepository(
        onFetch: (_) async => board,
        fallbackBoard: board,
        onResolveVoiceCommand: (_, __, ___) async =>
            const VoiceCommandResolution(
          action: 'task_complete_subject',
          reason: '提到了数学并表达完成',
          parserMode: 'rule_fallback',
          confidence: 0.88,
          normalizedTranscript: '数学订正好了',
          surface: VoiceCommandSurface.taskBoard,
          target: VoiceCommandTarget(subject: '数学'),
        ),
      );

      await tester.pumpWidget(
        StudyClawPadApp(
          autoLoad: true,
          repository: repository,
          speechRecognizer: _FakeSpeechRecognizer(
            transcript: const SpeechTranscript(
              transcript: '数学订正好了',
              locale: 'zh-CN',
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.byKey(const Key('voice-assistant-trigger')));
      await tester.pump();
      expect(find.text('结束说话'), findsOneWidget);
      await tester.tap(find.byKey(const Key('voice-assistant-trigger')));
      await tester.pumpAndSettle();

      expect(repository.taskGroupUpdates.length, 1);
      expect(repository.taskGroupUpdates.single.subject, '数学');
      expect(repository.taskGroupUpdates.single.groupTitle, isNull);
      expect(repository.taskGroupUpdates.single.completed, isTrue);
      expect(
        find.textContaining('已把“数学”学科任务标记为完成'),
        findsOneWidget,
      );
    });

    testWidgets('records transcript mode locally without resolving commands',
        (tester) async {
      final repository = _FakeTaskBoardRepository(
        onFetch: (_) async => _boardWithTasks(),
      );

      await tester.pumpWidget(
        StudyClawPadApp(
          autoLoad: true,
          repository: repository,
          speechRecognizer: _FakeSpeechRecognizer(
            transcript: const SpeechTranscript(
              transcript: '今天我先读第一段。然后我把第二段也读完了。',
              locale: 'zh-CN',
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.drag(find.byType(ListView).first, const Offset(0, -240));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('voice-mode-transcript')));
      await tester.pumpAndSettle();
      await tester.drag(find.byType(ListView).first, const Offset(0, -120));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('voice-scene-reading')));
      await tester.pumpAndSettle();

      await tester.tap(find.byKey(const Key('voice-assistant-trigger')));
      await tester.pump();
      expect(find.textContaining('正在记录：今天我先读第一段'), findsOneWidget);

      await tester.tap(find.byKey(const Key('voice-assistant-trigger')));
      await tester.pumpAndSettle();

      expect(repository.resolveVoiceCommandCalls, isEmpty);
      expect(find.byKey(const Key('voice-workbench-summary')), findsOneWidget);
      expect(find.textContaining('已按朗读场景整理成 2 段'), findsOneWidget);
      expect(find.textContaining('愿意一段一段读出来'), findsOneWidget);
      expect(find.text('第 1 段'), findsOneWidget);
      expect(find.text('第 2 段'), findsOneWidget);
    });

    testWidgets('analyzes poem recitation against reference text',
        (tester) async {
      final repository = _FakeTaskBoardRepository(
        onFetch: (_) async => _boardWithTasks(),
        onAnalyzeRecitation: (_, __, ___, ____, _____, ______) async =>
            const RecitationAnalysis(
          status: 'success',
          parserMode: 'llm_hybrid',
          scene: 'recitation',
          recognizedTitle: '江畔独步寻花',
          recognizedAuthor: '杜甫',
          referenceTitle: '江畔独步寻花',
          referenceAuthor: '杜甫',
          referenceText: '江畔独步寻花【唐】杜甫\n黄师塔前江水东，春光懒困倚微风。\n桃花一簇开无主，可爱深红爱浅红？',
          normalizedTranscript: '读办将办独步寻花糖杜甫黄思帕钳将水东春光染会以微风桃花一处开无主可爱深红爱浅红',
          reconstructedText: '江畔独步寻花 杜甫 黄师塔前江水东 春光懒困倚微风 桃花一簇开无主 可爱深红爱浅红',
          completionRatio: 0.78,
          needsRetry: true,
          summary: '已经识别为《江畔独步寻花》，主体内容对上了。',
          suggestion: '重点回看第 1 句，再完整背一遍。',
          issues: ['第 1 句还不够稳'],
          matchedLines: [
            RecitationLineAnalysis(
              index: 1,
              expected: '黄师塔前江水东，春光懒困倚微风。',
              observed: '黄思帕钳将水东春光染会以微风',
              matchRatio: 0.61,
              status: 'partial',
              notes: '主体对上，但有同音字替换',
            ),
            RecitationLineAnalysis(
              index: 2,
              expected: '桃花一簇开无主，可爱深红爱浅红？',
              observed: '桃花一处开无主可爱深红爱浅红',
              matchRatio: 0.94,
              status: 'matched',
              notes: '这一句整体比较稳',
            ),
          ],
        ),
      );

      await tester.pumpWidget(
        StudyClawPadApp(
          autoLoad: true,
          repository: repository,
          speechRecognizer: _FakeSpeechRecognizer(
            transcript: const SpeechTranscript(
              transcript: '读办将办独步寻花糖杜甫黄思帕钳将水东春光染会以微风桃花一处开无主可爱深红爱浅红',
              locale: 'zh-CN',
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.drag(find.byType(ListView).first, const Offset(0, -240));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('voice-mode-transcript')));
      await tester.pumpAndSettle();

      await tester.enterText(
        find.byKey(const Key('voice-reference-input')),
        '江畔独步寻花【唐】杜甫\n黄师塔前江水东，春光懒困倚微风。\n桃花一簇开无主，可爱深红爱浅红？',
      );
      await tester.pump();
      await tester.drag(find.byType(ListView).first, const Offset(0, 320));
      await tester.pumpAndSettle();

      await tester.tap(find.byKey(const Key('voice-assistant-trigger')));
      await tester.pump();
      await tester.tap(find.byKey(const Key('voice-assistant-trigger')));
      await tester.pumpAndSettle();

      expect(repository.recitationAnalysisCalls.length, 1);
      expect(
          find.byKey(const Key('voice-recitation-analysis')), findsOneWidget);
      expect(find.text('江畔独步寻花'), findsWidgets);
      expect(find.text('杜甫'), findsWidgets);
      expect(find.textContaining('重点回看第 1 句'), findsOneWidget);
      expect(find.textContaining('原文提示：需要家长/老师侧查看'), findsOneWidget);
      expect(find.textContaining('听到：黄思帕钳将水东春光染会以微风'), findsOneWidget);
    });

    testWidgets(
        'uses hidden task reference material automatically for recitation',
        (tester) async {
      const referenceText = '江畔独步寻花【唐】杜甫\n黄师塔前江水东，春光懒困倚微风。\n桃花一簇开无主，可爱深红爱浅红？';
      final board = _buildBoard(tasks: const [
        _TaskSeed(
          taskId: 1,
          subject: '语文',
          groupTitle: '古诗背诵',
          content: '背诵《江畔独步寻花》',
          completed: false,
          taskType: 'recitation',
          referenceTitle: '江畔独步寻花',
          referenceAuthor: '杜甫',
          referenceText: referenceText,
          referenceSource: 'extracted',
          hideReferenceFromChild: true,
          analysisMode: 'classical_poem',
        ),
      ]);
      final repository = _FakeTaskBoardRepository(
        onFetch: (_) async => board,
        onAnalyzeRecitation: (_, __, ___, ____, _____, ______) async =>
            const RecitationAnalysis(
          status: 'success',
          parserMode: 'llm_hybrid',
          scene: 'recitation',
          recognizedTitle: '江畔独步寻花',
          recognizedAuthor: '杜甫',
          referenceTitle: '江畔独步寻花',
          referenceAuthor: '杜甫',
          referenceText: referenceText,
          normalizedTranscript: '江畔独步寻花糖杜甫黄思塔前江水东春光缆会以微风桃花一处开无主可爱深红爱浅红',
          reconstructedText: '江畔独步寻花 杜甫 黄师塔前江水东 春光懒困倚微风 桃花一簇开无主 可爱深红爱浅红',
          completionRatio: 0.8,
          needsRetry: true,
          summary: '已经识别为《江畔独步寻花》，主体内容对上了。',
          suggestion: '建议把第一句再熟读一遍。',
          issues: ['第 1 句还不够稳'],
          matchedLines: [],
        ),
      );

      await tester.pumpWidget(
        StudyClawPadApp(
          autoLoad: true,
          repository: repository,
          speechRecognizer: _FakeSpeechRecognizer(
            transcript: const SpeechTranscript(
              transcript: '江畔独步寻花糖杜甫黄思塔前江水东春光缆会以微风桃花一处开无主可爱深红爱浅红',
              locale: 'zh-CN',
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.drag(find.byType(ListView).first, const Offset(0, -240));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('voice-mode-transcript')));
      await tester.pumpAndSettle();

      expect(find.byKey(const Key('voice-reference-input')), findsNothing);
      expect(
        find.byKey(const Key('voice-reference-task-summary')),
        findsOneWidget,
      );
      expect(find.textContaining('当前背诵任务：江畔独步寻花'), findsOneWidget);

      await tester.tap(find.byKey(const Key('voice-assistant-trigger')));
      await tester.pump();
      await tester.tap(find.byKey(const Key('voice-assistant-trigger')));
      await tester.pumpAndSettle();

      expect(repository.recitationAnalysisCalls.length, 1);
      expect(repository.recitationAnalysisCalls.first.referenceText,
          referenceText);
      expect(
          repository.recitationAnalysisCalls.first.metadata['reference_source'],
          'task');
      expect(
          repository
              .recitationAnalysisCalls.first.metadata['reference_task_source'],
          'extracted');
      expect(repository.recitationAnalysisCalls.first.metadata['task_id'], '1');
    });

    testWidgets('shows daily encouragement on the task board', (tester) async {
      final repository = _FakeTaskBoardRepository(
        onFetch: (_) async => _boardWithTasks(),
        onFetchDailyStats: (_) async => _dailyStats(
          completedTasks: 1,
          totalTasks: 3,
          encouragement: '今天已经迈出了第一步，继续一点点往前推进。',
        ),
      );

      await tester.pumpWidget(
        StudyClawPadApp(
          autoLoad: true,
          repository: repository,
        ),
      );
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(
        find.byKey(const Key('task-encouragement-card')),
        300,
        scrollable: find.byType(Scrollable).first,
      );
      await tester.pumpAndSettle();

      expect(find.byKey(const Key('task-encouragement-card')), findsOneWidget);
      expect(find.text('成长小鼓励'), findsOneWidget);
      expect(find.text('今天已经迈出了第一步，继续一点点往前推进。'), findsOneWidget);
      expect(find.text('已完成 1/3'), findsOneWidget);
    });

    testWidgets('auto-speaks task encouragement and supports replay',
        (tester) async {
      final speaker = _FakeWordSpeaker();
      final repository = _FakeTaskBoardRepository(
        onFetch: (_) async => _boardWithTasks(),
        onFetchDailyStats: (_) async => _dailyStats(
          completedTasks: 1,
          totalTasks: 3,
          encouragement: '今天已经迈出了第一步，继续一点点往前推进。',
        ),
      );
      final controller = WordPlaybackController(
        speaker: speaker,
        repository: repository,
      );

      await tester.pumpWidget(
        StudyClawPadApp(
          autoLoad: true,
          repository: repository,
          wordPlaybackController: controller,
        ),
      );
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(
        find.byKey(const Key('task-encouragement-card')),
        300,
        scrollable: find.byType(Scrollable).first,
      );
      await tester.pumpAndSettle();

      expect(speaker.spokenCalls.length, 1);
      expect(
        speaker.spokenCalls.single.text,
        contains('今天已经迈出了第一步，继续一点点往前推进。'),
      );

      await tester.tap(find.byKey(const Key('task-encouragement-replay')));
      await tester.pumpAndSettle();

      expect(speaker.spokenCalls.length, 2);
      expect(
        speaker.spokenCalls.last.text,
        contains('今天已经迈出了第一步，继续一点点往前推进。'),
      );
    });

    testWidgets('shows encouragement after completing a task', (tester) async {
      final initialBoard = _buildBoard(tasks: const [
        _TaskSeed(
          taskId: 1,
          subject: '英语',
          groupTitle: '默写',
          content: '默写 apple',
          completed: false,
        ),
        _TaskSeed(
          taskId: 2,
          subject: '英语',
          groupTitle: '默写',
          content: '默写 library',
          completed: false,
        ),
      ]);
      final updatedBoard = _buildBoard(tasks: const [
        _TaskSeed(
          taskId: 1,
          subject: '英语',
          groupTitle: '默写',
          content: '默写 apple',
          completed: true,
        ),
        _TaskSeed(
          taskId: 2,
          subject: '英语',
          groupTitle: '默写',
          content: '默写 library',
          completed: false,
        ),
      ]);
      final repository = _FakeTaskBoardRepository(
        onFetch: (_) async => initialBoard,
        onUpdateSingleTask: (_, __, ___) async => updatedBoard,
        onFetchDailyStats: (_) async => _dailyStats(
          completedTasks: 1,
          totalTasks: 2,
          encouragement: '今天已经完成 1 项任务，继续稳稳向前。',
        ),
      );

      await tester.pumpWidget(
        StudyClawPadApp(
          autoLoad: true,
          repository: repository,
        ),
      );
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(
        find.text('默写 apple'),
        300,
        scrollable: find.byType(Scrollable).first,
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('默写 apple'));
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(
        find.byKey(const Key('task-encouragement-card')),
        300,
        scrollable: find.byType(Scrollable).first,
      );
      await tester.pumpAndSettle();

      expect(repository.singleTaskUpdates.length, 1);
      expect(find.byKey(const Key('task-encouragement-card')), findsOneWidget);
      expect(find.text('这一步不轻松，你还是认真拿下了。现在已经完成 1/2 项，开了个好头，继续保持。'),
          findsOneWidget);
    });
  });

  group('WordPlayback encouragement', () {
    test('speaks coach message with a warmer voice tone', () async {
      final speaker = _FakeWordSpeaker();
      final controller = WordPlaybackController(
        speaker: speaker,
        repository: _FakeTaskBoardRepository(
          onFetch: (_) async => _boardWithTasks(),
        ),
      );

      await controller.speakCoachMessage('这一步完成啦，继续保持。');

      expect(speaker.spokenCalls.length, 1);
      expect(speaker.spokenCalls.single.language, WordPlaybackLanguage.chinese);
      expect(speaker.spokenCalls.single.speechRate, 0.44);
      expect(speaker.spokenCalls.single.pitch, 1.08);
      expect(
        speaker.spokenCalls.single.text,
        contains('我会继续陪着你，我们慢慢来。'),
      );
    });

    test('shows encouragement when dictation reaches the end', () async {
      const startSession = DictationSession(
        sessionId: 'session_finish_001',
        wordListId: 'word_list_finish_001',
        status: 'active',
        currentIndex: 0,
        totalItems: 1,
        playedCount: 0,
        completedItems: 0,
        currentItem: WordItem(index: 1, text: 'apple', meaning: '苹果'),
        gradingStatus: 'idle',
      );
      const completedSession = DictationSession(
        sessionId: 'session_finish_001',
        wordListId: 'word_list_finish_001',
        status: 'completed',
        currentIndex: 1,
        totalItems: 1,
        playedCount: 1,
        completedItems: 1,
        gradingStatus: 'idle',
      );
      final repository = _FakeTaskBoardRepository(
        onFetch: (_) async => _boardWithTasks(),
        onStartDictation: (_) async => startSession,
        onNextDictationSession: (_, __) async => completedSession,
      );
      final controller = WordPlaybackController(
        speaker: _FakeWordSpeaker(),
        repository: repository,
      );
      const request = TaskBoardRequest(
        apiBaseUrl: 'http://localhost:8080',
        familyId: 306,
        userId: 1,
        date: '2026-03-12',
      );

      await controller.startDictation(request);
      await controller.nextWord(request.apiBaseUrl);

      expect(controller.state.noticeMessage, '这一组单词都完成啦，你坚持到了最后！');
    });
  });
}

class _FakeTaskBoardRepository implements TaskBoardRepository {
  _FakeTaskBoardRepository(
      {required this.onFetch,
      this.onFetchDailyStats,
      this.onUpdateSingleTask,
      this.onFetchWordList,
      this.onStartDictation,
      this.onNextDictationSession,
      this.onGetDictationSession,
      this.onGradeDictationSession,
      this.onResolveVoiceCommand,
      this.onAnalyzeRecitation,
      TaskBoard? fallbackBoard})
      : _lastBoard = fallbackBoard ?? _boardWithTasks();

  final Future<TaskBoard> Function(TaskBoardRequest request) onFetch;
  final Future<DailyStats> Function(TaskBoardRequest request)?
      onFetchDailyStats;
  final Future<TaskBoard> Function(
    TaskBoardRequest request,
    int taskId,
    bool completed,
  )? onUpdateSingleTask;
  final Future<WordList> Function(TaskBoardRequest request)? onFetchWordList;
  final Future<DictationSession> Function(TaskBoardRequest request)?
      onStartDictation;
  final Future<DictationSession> Function(String sessionId, String apiBaseUrl)?
      onNextDictationSession;
  final Future<DictationSession> Function(String sessionId, String apiBaseUrl)?
      onGetDictationSession;
  final Future<DictationSession> Function(
      String sessionId,
      String apiBaseUrl,
      String photoBase64,
      String language,
      String mode)? onGradeDictationSession;
  final Future<VoiceCommandResolution> Function(
    TaskBoardRequest request,
    String transcript,
    VoiceCommandContext context,
  )? onResolveVoiceCommand;
  final Future<RecitationAnalysis> Function(
    String apiBaseUrl,
    String transcript,
    String scene,
    String? locale,
    String? referenceText,
    Map<String, String> metadata,
  )? onAnalyzeRecitation;

  final List<_SingleTaskUpdateCall> singleTaskUpdates = [];
  final List<_TaskGroupUpdateCall> taskGroupUpdates = [];
  final List<String> nextDictationCalls = [];
  final List<String> resolveVoiceCommandCalls = [];
  final List<_RecitationAnalysisCall> recitationAnalysisCalls = [];
  final TaskBoard _lastBoard;

  @override
  Future<TaskBoard> fetchBoard(TaskBoardRequest request) async =>
      await onFetch(request);
  @override
  Future<DailyStats> fetchDailyStats(TaskBoardRequest request) async =>
      await (onFetchDailyStats?.call(request) ?? Future.value(_dailyStats()));
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
    return await (onUpdateSingleTask?.call(request, taskId, completed) ??
        Future.value(_lastBoard));
  }

  @override
  Future<TaskBoard> updateTaskGroup(TaskBoardRequest request,
      {required String subject,
      String? groupTitle,
      required bool completed}) async {
    taskGroupUpdates.add(_TaskGroupUpdateCall(
      subject: subject,
      groupTitle: groupTitle,
      completed: completed,
    ));
    return _lastBoard;
  }

  @override
  Future<TaskBoard> updateAllTasks(TaskBoardRequest request,
          {required bool completed}) async =>
      _lastBoard;
  @override
  Future<WordList> fetchWordList(TaskBoardRequest request) async =>
      await (onFetchWordList?.call(request) ??
          Future.value(const WordList(
              wordListId: '0',
              familyId: 0,
              childId: 0,
              assignedDate: '',
              title: '',
              language: WordPlaybackLanguage.english,
              items: [],
              totalItems: 0)));
  @override
  Future<DictationSession> startDictationSession(
          TaskBoardRequest request) async =>
      await (onStartDictation?.call(request) ?? Future.value(_emptySession));
  @override
  Future<DictationSession> nextDictationSession(
      String sessionId, String apiBaseUrl) async {
    nextDictationCalls.add(sessionId);
    return await (onNextDictationSession?.call(sessionId, apiBaseUrl) ??
        Future.value(_emptySession));
  }

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

  @override
  Future<VoiceCommandResolution> resolveVoiceCommand(TaskBoardRequest request,
      {required String transcript,
      required VoiceCommandContext context}) async {
    resolveVoiceCommandCalls.add(transcript);
    return await (onResolveVoiceCommand?.call(request, transcript, context) ??
        Future.value(const VoiceCommandResolution(
          action: 'none',
          reason: '未命中',
          parserMode: 'rule_fallback',
          confidence: 0,
          normalizedTranscript: '',
          surface: VoiceCommandSurface.taskBoard,
          target: VoiceCommandTarget(),
        )));
  }

  @override
  Future<RecitationAnalysis> analyzeRecitation(
    String apiBaseUrl, {
    required String transcript,
    required String scene,
    String? locale,
    String? referenceText,
    Map<String, String> metadata = const <String, String>{},
  }) async {
    recitationAnalysisCalls.add(_RecitationAnalysisCall(
      transcript: transcript,
      scene: scene,
      locale: locale,
      referenceText: referenceText ?? '',
      metadata: Map<String, String>.from(metadata),
    ));
    return await (onAnalyzeRecitation?.call(
          apiBaseUrl,
          transcript,
          scene,
          locale,
          referenceText,
          metadata,
        ) ??
        Future.value(const RecitationAnalysis(
          status: 'success',
          parserMode: 'rule_fallback',
          scene: 'recitation',
          recognizedTitle: '',
          recognizedAuthor: '',
          referenceTitle: '',
          referenceAuthor: '',
          referenceText: '',
          normalizedTranscript: '',
          reconstructedText: '',
          completionRatio: 0,
          needsRetry: true,
          summary: '缺少参考原文，暂时无法比对。',
          suggestion: '请补充原文后重试。',
          issues: [],
          matchedLines: [],
        )));
  }
}

class _SingleTaskUpdateCall {
  const _SingleTaskUpdateCall({required this.taskId, required this.completed});
  final int taskId;
  final bool completed;
}

class _TaskGroupUpdateCall {
  const _TaskGroupUpdateCall({
    required this.subject,
    required this.groupTitle,
    required this.completed,
  });

  final String subject;
  final String? groupTitle;
  final bool completed;
}

class _RecitationAnalysisCall {
  const _RecitationAnalysisCall({
    required this.transcript,
    required this.scene,
    required this.locale,
    required this.referenceText,
    required this.metadata,
  });

  final String transcript;
  final String scene;
  final String? locale;
  final String referenceText;
  final Map<String, String> metadata;
}

TaskBoard _buildBoard({required List<_TaskSeed> tasks}) {
  final subjectCounts = <String, int>{};
  final subjectCompletedCounts = <String, int>{};
  final groupCounts = <String, int>{};
  final groupCompletedCounts = <String, int>{};

  for (final task in tasks) {
    subjectCounts.update(task.subject, (value) => value + 1, ifAbsent: () => 1);
    if (task.completed) {
      subjectCompletedCounts.update(
        task.subject,
        (value) => value + 1,
        ifAbsent: () => 1,
      );
    }

    final groupKey = '${task.subject}::${task.groupTitle}';
    groupCounts.update(groupKey, (value) => value + 1, ifAbsent: () => 1);
    if (task.completed) {
      groupCompletedCounts.update(
        groupKey,
        (value) => value + 1,
        ifAbsent: () => 1,
      );
    }
  }

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
              status: t.completed ? 'completed' : 'pending',
              taskType: t.taskType,
              referenceTitle: t.referenceTitle,
              referenceAuthor: t.referenceAuthor,
              referenceText: t.referenceText,
              referenceSource: t.referenceSource,
              hideReferenceFromChild: t.hideReferenceFromChild,
              analysisMode: t.analysisMode))
          .toList(),
      groups: subjectCounts.entries.map((entry) {
        final completed = subjectCompletedCounts[entry.key] ?? 0;
        final total = entry.value;
        return TaskGroup(
          subject: entry.key,
          total: total,
          completed: completed,
          pending: total - completed,
          status: completed == total
              ? 'completed'
              : completed == 0
                  ? 'pending'
                  : 'partial',
        );
      }).toList(),
      homeworkGroups: groupCounts.entries.map((entry) {
        final separatorIndex = entry.key.indexOf('::');
        final subject = entry.key.substring(0, separatorIndex);
        final groupTitle = entry.key.substring(separatorIndex + 2);
        final completed = groupCompletedCounts[entry.key] ?? 0;
        final total = entry.value;
        return HomeworkGroup(
          subject: subject,
          groupTitle: groupTitle,
          total: total,
          completed: completed,
          pending: total - completed,
          status: completed == total
              ? 'completed'
              : completed == 0
                  ? 'pending'
                  : 'partial',
        );
      }).toList(),
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

DailyStats _dailyStats({
  int completedTasks = 0,
  int totalTasks = 0,
  String encouragement = '',
}) {
  return DailyStats(
    period: 'daily',
    startDate: '2026-03-12',
    endDate: '2026-03-12',
    encouragement: encouragement,
    totals: StatsTotals(
      totalTasks: totalTasks,
      completedTasks: completedTasks,
      pendingTasks: totalTasks - completedTasks,
      completionRate: totalTasks == 0 ? 0 : completedTasks / totalTasks,
      autoPoints: 0,
      manualPoints: 0,
      totalPointsDelta: 0,
      pointsBalance: 0,
      wordItems: 0,
      completedWordItems: 0,
      dictationSessions: 0,
    ),
  );
}

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
      required this.completed,
      this.taskType = '',
      this.referenceTitle = '',
      this.referenceAuthor = '',
      this.referenceText = '',
      this.referenceSource = '',
      this.hideReferenceFromChild = false,
      this.analysisMode = ''});
  final int taskId;
  final String subject;
  final String groupTitle;
  final String content;
  final bool completed;
  final String taskType;
  final String referenceTitle;
  final String referenceAuthor;
  final String referenceText;
  final String referenceSource;
  final bool hideReferenceFromChild;
  final String analysisMode;
}

class _FakeWordSpeaker implements WordSpeaker {
  _FakeWordSpeaker();
  final List<_FakeSpokenCall> spokenCalls = [];

  @override
  bool get supportsPlayback => true;

  @override
  Future<void> speak(
    String text, {
    required WordPlaybackLanguage language,
    double? speechRate,
    double? pitch,
  }) async {
    spokenCalls.add(_FakeSpokenCall(
      text: text,
      language: language,
      speechRate: speechRate,
      pitch: pitch,
    ));
  }

  @override
  Future<void> stop() async {}
}

class _FakeSpokenCall {
  const _FakeSpokenCall({
    required this.text,
    required this.language,
    this.speechRate,
    this.pitch,
  });

  final String text;
  final WordPlaybackLanguage language;
  final double? speechRate;
  final double? pitch;
}

class _FakeSpeechRecognizer implements SpeechRecognizer {
  _FakeSpeechRecognizer({
    required this.transcript,
  });

  final SpeechTranscript transcript;
  bool _isListening = false;

  @override
  bool get supportsRecognition => true;

  @override
  bool get isListening => _isListening;

  @override
  Future<void> startListening({
    required String locale,
    SpeechTranscriptListener? onTranscriptChanged,
    SpeechSegmentListener? onSegmentCommitted,
  }) async {
    _isListening = true;
    onTranscriptChanged?.call(transcript.transcript);
    onSegmentCommitted?.call(transcript.transcript);
  }

  @override
  Future<SpeechTranscript> finishListening() async {
    _isListening = false;
    return transcript;
  }

  @override
  Future<SpeechTranscript> listenOnce({required String locale}) async =>
      transcript;

  @override
  Future<void> stop() async {
    _isListening = false;
  }
}
