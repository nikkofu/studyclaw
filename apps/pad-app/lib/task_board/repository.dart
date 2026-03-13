import 'package:pad_app/task_board/api_client.dart';
import 'package:pad_app/task_board/models.dart';
import 'package:pad_app/task_board/recitation_analysis.dart';
import 'package:pad_app/task_board/weekly_stats.dart';
import 'package:pad_app/task_board/daily_stats.dart';
import 'package:pad_app/voice_commands/models.dart';
import 'package:pad_app/word_playback/models.dart';

abstract interface class TaskBoardRepository {
  Future<TaskBoard> fetchBoard(TaskBoardRequest request);

  Future<WeeklyStats> fetchWeeklyStats(TaskBoardRequest request);

  Future<DailyStats> fetchDailyStats(TaskBoardRequest request);

  Future<Map<String, dynamic>> fetchMonthlyStats(TaskBoardRequest request);

  Future<TaskBoard> updateSingleTask(
    TaskBoardRequest request, {
    required int taskId,
    required bool completed,
  });

  Future<TaskBoard> updateTaskGroup(
    TaskBoardRequest request, {
    required String subject,
    String? groupTitle,
    required bool completed,
  });

  Future<TaskBoard> updateAllTasks(
    TaskBoardRequest request, {
    required bool completed,
  });

  Future<WordList> fetchWordList(TaskBoardRequest request);

  Future<DictationSession> startDictationSession(TaskBoardRequest request);

  Future<DictationSession> getDictationSession(
      String sessionId, String apiBaseUrl);

  Future<DictationSession> replayDictationSession(
      String sessionId, String apiBaseUrl);

  Future<DictationSession> nextDictationSession(
      String sessionId, String apiBaseUrl);

  Future<DictationSession> previousDictationSession(
      String sessionId, String apiBaseUrl);

  Future<DictationSession> gradeDictationSession({
    required String sessionId,
    required String apiBaseUrl,
    required String photoBase64,
    required String language,
    required String mode,
  });

  Future<VoiceCommandResolution> resolveVoiceCommand(
    TaskBoardRequest request, {
    required String transcript,
    required VoiceCommandContext context,
  });

  Future<RecitationAnalysis> analyzeRecitation(
    String apiBaseUrl, {
    required String transcript,
    required String scene,
    String? locale,
    String? referenceText,
    Map<String, String> metadata = const <String, String>{},
  });
}

class RemoteTaskBoardRepository implements TaskBoardRepository {
  const RemoteTaskBoardRepository({
    this.clientFactory = defaultTaskBoardApiClientFactory,
  });

  final TaskBoardApiClientFactory clientFactory;

  @override
  Future<TaskBoard> fetchBoard(TaskBoardRequest request) async {
    final payload = await _clientFor(request).send(
      'GET',
      '/api/v1/tasks',
      query: {
        'family_id': '${request.familyId}',
        'user_id': '${request.userId}',
        'date': request.date,
      },
    );
    return TaskBoard.fromJson(payload);
  }

  @override
  Future<WeeklyStats> fetchWeeklyStats(TaskBoardRequest request) async {
    final payload = await _clientFor(request).send(
      'GET',
      '/api/v1/stats/weekly',
      query: {
        'family_id': '${request.familyId}',
        'user_id': '${request.userId}',
        'end_date': request.date,
      },
    );
    return WeeklyStats.fromJson(payload);
  }

  @override
  Future<DailyStats> fetchDailyStats(TaskBoardRequest request) async {
    final payload = await _clientFor(request).send(
      'GET',
      '/api/v1/stats/daily',
      query: {
        'family_id': '${request.familyId}',
        'user_id': '${request.userId}',
        'date': request.date,
      },
    );
    return DailyStats.fromJson(payload);
  }

  @override
  Future<Map<String, dynamic>> fetchMonthlyStats(
      TaskBoardRequest request) async {
    final payload = await _clientFor(request).send(
      'GET',
      '/api/v1/stats/monthly',
      query: {
        'family_id': '${request.familyId}',
        'user_id': '${request.userId}',
        'end_date': request.date,
      },
    );
    return payload;
  }

  @override
  Future<TaskBoard> updateSingleTask(
    TaskBoardRequest request, {
    required int taskId,
    required bool completed,
  }) async {
    final payload = await _clientFor(request).send(
      'PATCH',
      '/api/v1/tasks/status/item',
      body: {
        'family_id': request.familyId,
        'assignee_id': request.userId,
        'task_id': taskId,
        'completed': completed,
        'assigned_date': request.date,
      },
    );
    return TaskBoard.fromJson(payload);
  }

  @override
  Future<TaskBoard> updateTaskGroup(
    TaskBoardRequest request, {
    required String subject,
    String? groupTitle,
    required bool completed,
  }) async {
    final payload = await _clientFor(request).send(
      'PATCH',
      '/api/v1/tasks/status/group',
      body: {
        'family_id': request.familyId,
        'assignee_id': request.userId,
        'subject': subject,
        if (groupTitle != null && groupTitle.isNotEmpty)
          'group_title': groupTitle,
        'completed': completed,
        'assigned_date': request.date,
      },
    );
    return TaskBoard.fromJson(payload);
  }

  @override
  Future<TaskBoard> updateAllTasks(
    TaskBoardRequest request, {
    required bool completed,
  }) async {
    final payload = await _clientFor(request).send(
      'PATCH',
      '/api/v1/tasks/status/all',
      body: {
        'family_id': request.familyId,
        'assignee_id': request.userId,
        'completed': completed,
        'assigned_date': request.date,
      },
    );
    return TaskBoard.fromJson(payload);
  }

  @override
  Future<WordList> fetchWordList(TaskBoardRequest request) async {
    final payload = await _clientFor(request).send(
      'GET',
      '/api/v1/word-lists',
      query: {
        'family_id': '${request.familyId}',
        'child_id': '${request.userId}',
        'date': request.date,
      },
    );
    return WordList.fromJson(payload['word_list'] as Map<String, dynamic>);
  }

  @override
  Future<DictationSession> startDictationSession(
      TaskBoardRequest request) async {
    final payload = await _clientFor(request).send(
      'POST',
      '/api/v1/dictation-sessions/start',
      body: {
        'family_id': request.familyId,
        'child_id': request.userId,
        'assigned_date': request.date,
      },
    );
    return DictationSession.fromJson(
        payload['dictation_session'] as Map<String, dynamic>);
  }

  @override
  Future<DictationSession> getDictationSession(
      String sessionId, String apiBaseUrl) async {
    final payload = await clientFactory(apiBaseUrl).send(
      'GET',
      '/api/v1/dictation-sessions/$sessionId',
    );
    return DictationSession.fromJson(
        payload['dictation_session'] as Map<String, dynamic>);
  }

  @override
  Future<DictationSession> replayDictationSession(
      String sessionId, String apiBaseUrl) async {
    final payload = await clientFactory(apiBaseUrl).send(
      'POST',
      '/api/v1/dictation-sessions/$sessionId/replay',
    );
    return DictationSession.fromJson(
        payload['dictation_session'] as Map<String, dynamic>);
  }

  @override
  Future<DictationSession> nextDictationSession(
      String sessionId, String apiBaseUrl) async {
    final payload = await clientFactory(apiBaseUrl).send(
      'POST',
      '/api/v1/dictation-sessions/$sessionId/next',
    );
    return DictationSession.fromJson(
        payload['dictation_session'] as Map<String, dynamic>);
  }

  @override
  Future<DictationSession> previousDictationSession(
      String sessionId, String apiBaseUrl) async {
    final payload = await clientFactory(apiBaseUrl).send(
      'POST',
      '/api/v1/dictation-sessions/$sessionId/prev',
    );
    return DictationSession.fromJson(
        payload['dictation_session'] as Map<String, dynamic>);
  }

  @override
  Future<DictationSession> gradeDictationSession({
    required String sessionId,
    required String apiBaseUrl,
    required String photoBase64,
    required String language,
    required String mode,
  }) async {
    final client = clientFactory(apiBaseUrl);
    final payload = await client.send(
      'POST',
      '/api/v1/dictation-sessions/$sessionId/grade',
      body: {
        'photo': photoBase64,
        'language': language,
        'mode': mode,
      },
    );
    return DictationSession.fromJson(
        payload['dictation_session'] as Map<String, dynamic>);
  }

  @override
  Future<VoiceCommandResolution> resolveVoiceCommand(
    TaskBoardRequest request, {
    required String transcript,
    required VoiceCommandContext context,
  }) async {
    final payload = await _clientFor(request).send(
      'POST',
      '/api/v1/voice-commands/resolve',
      body: {
        'transcript': transcript,
        'context': context.toJson(),
      },
    );
    return VoiceCommandResolution.fromJson(
      payload['resolution'] as Map<String, dynamic>,
    );
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
    final payload = await clientFactory(apiBaseUrl).send(
      'POST',
      '/api/v1/recitation/analyze',
      body: {
        'transcript': transcript,
        'scene': scene,
        if (locale != null && locale.trim().isNotEmpty) 'locale': locale,
        if (referenceText != null && referenceText.trim().isNotEmpty)
          'reference_text': referenceText,
        if (metadata.isNotEmpty) 'metadata': metadata,
      },
    );
    return RecitationAnalysis.fromJson(
      payload['analysis'] as Map<String, dynamic>,
    );
  }

  TaskBoardApiClient _clientFor(TaskBoardRequest request) {
    return clientFactory(request.apiBaseUrl);
  }
}
