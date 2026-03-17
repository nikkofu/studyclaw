enum WordPlaybackLanguage { english, chinese }

enum WordPlaybackMode { word, meaning }

extension WordPlaybackModeX on WordPlaybackMode {
  String get label {
    switch (this) {
      case WordPlaybackMode.word:
        return '播报单词';
      case WordPlaybackMode.meaning:
        return '播报释义';
    }
  }
}

extension WordPlaybackLanguageX on WordPlaybackLanguage {
  String get label {
    switch (this) {
      case WordPlaybackLanguage.english:
        return '英语';
      case WordPlaybackLanguage.chinese:
        return '语文';
    }
  }

  String get localeCode {
    switch (this) {
      case WordPlaybackLanguage.english:
        return 'en-US';
      case WordPlaybackLanguage.chinese:
        return 'zh-CN';
    }
  }

  String get hintText {
    switch (this) {
      case WordPlaybackLanguage.english:
        return '每行一个单词，例如：apple';
      case WordPlaybackLanguage.chinese:
        return '每行一个词语，例如：认真';
    }
  }

  List<String> get sampleWords {
    switch (this) {
      case WordPlaybackLanguage.english:
        return const <String>['apple', 'library', 'notebook', 'tomorrow'];
      case WordPlaybackLanguage.chinese:
        return const <String>['认真', '练习', '进步', '勇敢'];
    }
  }

  static WordPlaybackLanguage fromString(String value) {
    if (value.toLowerCase().contains('chinese') || value.contains('语文')) {
      return WordPlaybackLanguage.chinese;
    }
    return WordPlaybackLanguage.english;
  }
}

class WordItem {
  const WordItem({
    required this.index,
    required this.text,
    this.meaning,
    this.hint,
  });

  factory WordItem.fromJson(Map<String, dynamic> json) {
    return WordItem(
      index: _readInt(json['index']),
      text: json['text']?.toString() ?? '',
      meaning: json['meaning']?.toString(),
      hint: json['hint']?.toString(),
    );
  }

  final int index;
  final String text;
  final String? meaning;
  final String? hint;
}

class WordList {
  const WordList({
    required this.wordListId,
    required this.familyId,
    required this.childId,
    required this.assignedDate,
    required this.title,
    required this.language,
    required this.items,
    required this.totalItems,
  });

  factory WordList.fromJson(Map<String, dynamic> json) {
    return WordList(
      wordListId: json['word_list_id']?.toString() ?? '',
      familyId: _readInt(json['family_id']),
      childId: _readInt(json['child_id']),
      assignedDate: json['assigned_date']?.toString() ?? '',
      title: json['title']?.toString() ?? '',
      language: WordPlaybackLanguageX.fromString(
          json['language']?.toString() ?? 'english'),
      items: ((json['items'] as List<dynamic>? ?? const <dynamic>[]))
          .map((item) => WordItem.fromJson(item as Map<String, dynamic>))
          .toList(),
      totalItems: _readInt(json['total_items']),
    );
  }

  final String wordListId;
  final int familyId;
  final int childId;
  final String assignedDate;
  final String title;
  final WordPlaybackLanguage language;
  final List<WordItem> items;
  final int totalItems;
}

class DictationSession {
  const DictationSession({
    required this.sessionId,
    required this.wordListId,
    required this.status,
    required this.currentIndex,
    required this.totalItems,
    required this.playedCount,
    required this.completedItems,
    this.currentItem,
    required this.gradingStatus,
    this.gradingError,
    this.gradingRequestedAt,
    this.gradingCompletedAt,
    this.gradingResult,
    this.debugContext,
    this.startedAt,
    this.updatedAt,
  });

  factory DictationSession.fromJson(Map<String, dynamic> json) {
    return DictationSession(
      sessionId: json['session_id']?.toString() ?? '',
      wordListId: json['word_list_id']?.toString() ?? '',
      status: json['status']?.toString() ?? 'active',
      currentIndex: _readInt(json['current_index']),
      totalItems: _readInt(json['total_items']),
      playedCount: _readInt(json['played_count']),
      completedItems: _readInt(json['completed_items']),
      currentItem: json['current_item'] != null
          ? WordItem.fromJson(json['current_item'] as Map<String, dynamic>)
          : null,
      gradingStatus: json['grading_status']?.toString() ?? 'idle',
      gradingError: json['grading_error']?.toString(),
      gradingRequestedAt: json['grading_requested_at']?.toString(),
      gradingCompletedAt: json['grading_completed_at']?.toString(),
      gradingResult: json['grading_result'] != null
          ? DictationGradingResult.fromJson(
              json['grading_result'] as Map<String, dynamic>)
          : null,
      debugContext: json['debug_context'] != null
          ? DictationDebugContext.fromJson(
              json['debug_context'] as Map<String, dynamic>)
          : null,
      startedAt: json['started_at']?.toString(),
      updatedAt: json['updated_at']?.toString(),
    );
  }

  final String sessionId;
  final String wordListId;
  final String status;
  final int currentIndex;
  final int totalItems;
  final int playedCount;
  final int completedItems;
  final WordItem? currentItem;
  final String gradingStatus;
  final String? gradingError;
  final String? gradingRequestedAt;
  final String? gradingCompletedAt;
  final DictationGradingResult? gradingResult;
  final DictationDebugContext? debugContext;
  final String? startedAt;
  final String? updatedAt;

  bool get isCompleted => status == 'completed';
  bool get isGradingPending =>
      gradingStatus == 'pending' || gradingStatus == 'processing';
  bool get hasGradingResult =>
      gradingStatus == 'completed' && gradingResult != null;
  bool get isShowingResultView => isGradingPending || hasGradingResult;
  bool get isWaitingForGradingView => isGradingPending && !hasGradingResult;
}

class DictationDebugContext {
  const DictationDebugContext({
    this.photoSha1,
    this.photoBytes = 0,
    this.language,
    this.mode,
    this.workerStage,
    this.logFile,
    this.logKeywords = const <String>[],
  });

  factory DictationDebugContext.fromJson(Map<String, dynamic> json) {
    return DictationDebugContext(
      photoSha1: json['photo_sha1']?.toString(),
      photoBytes: _readInt(json['photo_bytes']),
      language: json['language']?.toString(),
      mode: json['mode']?.toString(),
      workerStage: json['worker_stage']?.toString(),
      logFile: json['log_file']?.toString(),
      logKeywords: (json['log_keywords'] as List<dynamic>? ?? const <dynamic>[])
          .map((item) => item.toString())
          .where((item) => item.trim().isNotEmpty)
          .toList(),
    );
  }

  final String? photoSha1;
  final int photoBytes;
  final String? language;
  final String? mode;
  final String? workerStage;
  final String? logFile;
  final List<String> logKeywords;
}

class DictationStageMeta {
  const DictationStageMeta({
    required this.key,
    required this.label,
    required this.hint,
  });

  final String key;
  final String label;
  final String hint;
}

String resolveDictationWorkerStage(DictationSession? session) {
  final workerStage = session?.debugContext?.workerStage?.trim() ?? '';
  if (workerStage.isNotEmpty) {
    return workerStage;
  }

  switch (session?.gradingStatus) {
    case 'pending':
      return 'queued';
    case 'processing':
      return 'processing';
    case 'completed':
      return 'completed';
    case 'failed':
      return 'failed';
    default:
      return 'idle';
  }
}

DictationStageMeta describeDictationStage(DictationSession? session) {
  switch (resolveDictationWorkerStage(session)) {
    case 'queued':
      return const DictationStageMeta(
        key: 'queued',
        label: '云端排队',
        hint: '照片已经到云端，马上轮到 AI 来检查。',
      );
    case 'processing':
      return const DictationStageMeta(
        key: 'processing',
        label: '准备批改',
        hint: '后台已经接手，正在整理这次交卷内容。',
      );
    case 'loading_word_list':
      return const DictationStageMeta(
        key: 'loading_word_list',
        label: '准备答案',
        hint: '正在把正确答案和这次任务配对好。',
      );
    case 'llm_grading':
      return const DictationStageMeta(
        key: 'llm_grading',
        label: 'AI 比对',
        hint: 'AI 正在认真看照片，并和正确答案一项项比对。',
      );
    case 'completed':
      return const DictationStageMeta(
        key: 'completed',
        label: '结果同步',
        hint: '批改完成，结果已经回到平板。',
      );
    case 'mark_processing_failed':
      return const DictationStageMeta(
        key: 'mark_processing_failed',
        label: '接单受阻',
        hint: '照片已经上传，但后台没有顺利进入批改流程。',
      );
    case 'load_word_list_failed':
      return const DictationStageMeta(
        key: 'load_word_list_failed',
        label: '答案没准备好',
        hint: '后台准备正确答案时出了点问题。',
      );
    case 'llm_grading_failed':
      return const DictationStageMeta(
        key: 'llm_grading_failed',
        label: 'AI 没看清',
        hint: 'AI 这次没有顺利完成比对，可以重新拍照再试一次。',
      );
    case 'persist_result_failed':
      return const DictationStageMeta(
        key: 'persist_result_failed',
        label: '结果回传受阻',
        hint: 'AI 已经看完了，但结果回到平板时出了点问题。',
      );
    case 'failed':
      return const DictationStageMeta(
        key: 'failed',
        label: '这次受阻了',
        hint: '后台没有顺利完成批改，可以重新拍照交卷。',
      );
    default:
      return const DictationStageMeta(
        key: 'idle',
        label: '等待交卷',
        hint: '拍一张清楚的作业照片，交给 AI 批改吧。',
      );
  }
}

class DictationGradingResult {
  const DictationGradingResult({
    required this.gradingId,
    required this.status,
    required this.score,
    required this.gradedItems,
    required this.aiFeedback,
    required this.createdAt,
    this.annotatedPhotoUrl,
    this.annotatedPhotoWidth = 0,
    this.annotatedPhotoHeight = 0,
    this.markRegions = const <DictationMarkRegion>[],
  });

  factory DictationGradingResult.fromJson(Map<String, dynamic> json) {
    return DictationGradingResult(
      gradingId: json['grading_id']?.toString() ?? '',
      status: json['status']?.toString() ?? 'passed',
      score: _readInt(json['score']),
      gradedItems:
          ((json['graded_items'] as List<dynamic>? ?? const <dynamic>[]))
              .map((item) =>
                  DictationGradedItem.fromJson(item as Map<String, dynamic>))
              .toList(),
      aiFeedback: json['ai_feedback']?.toString() ?? '',
      createdAt: json['created_at']?.toString() ?? '',
      annotatedPhotoUrl: json['annotated_photo_url']?.toString(),
      annotatedPhotoWidth: _readInt(json['annotated_photo_width']),
      annotatedPhotoHeight: _readInt(json['annotated_photo_height']),
      markRegions: ((json['mark_regions'] as List<dynamic>? ?? const <dynamic>[]))
          .map((item) =>
              DictationMarkRegion.fromJson(item as Map<String, dynamic>))
          .toList(),
    );
  }

  final String gradingId;
  final String status;
  final int score;
  final List<DictationGradedItem> gradedItems;
  final String aiFeedback;
  final String createdAt;
  final String? annotatedPhotoUrl;
  final int annotatedPhotoWidth;
  final int annotatedPhotoHeight;
  final List<DictationMarkRegion> markRegions;

  int get incorrectCount => gradedItems
      .where((item) => !item.isCorrect || item.needsCorrection)
      .length;

  bool get hasAnnotatedPhoto => (annotatedPhotoUrl?.trim().isNotEmpty ?? false);
  bool get hasMarkRegions => markRegions.isNotEmpty;
}

class DictationMarkRegion {
  const DictationMarkRegion({
    required this.index,
    required this.isCorrect,
    required this.left,
    required this.top,
    required this.width,
    required this.height,
    this.expected,
    this.actual,
    this.markerLabel,
  });

  factory DictationMarkRegion.fromJson(Map<String, dynamic> json) {
    return DictationMarkRegion(
      index: _readInt(json['index']),
      expected: json['expected']?.toString(),
      actual: json['actual']?.toString(),
      isCorrect: json['is_correct'] == true,
      left: _readDouble(json['left']),
      top: _readDouble(json['top']),
      width: _readDouble(json['width']),
      height: _readDouble(json['height']),
      markerLabel: json['marker_label']?.toString(),
    );
  }

  final int index;
  final String? expected;
  final String? actual;
  final bool isCorrect;
  final double left;
  final double top;
  final double width;
  final double height;
  final String? markerLabel;
}

class DictationGradedItem {
  const DictationGradedItem({
    required this.index,
    required this.expected,
    required this.actual,
    required this.isCorrect,
    required this.needsCorrection,
    this.meaning,
    this.comment,
  });

  factory DictationGradedItem.fromJson(Map<String, dynamic> json) {
    return DictationGradedItem(
      index: _readInt(json['index']),
      expected: json['expected']?.toString() ?? '',
      actual: json['actual']?.toString() ?? '',
      isCorrect: json['is_correct'] == true,
      needsCorrection: json['needs_correction'] == true,
      meaning: json['meaning']?.toString(),
      comment: json['comment']?.toString(),
    );
  }

  final int index;
  final String expected;
  final String actual;
  final bool isCorrect;
  final bool needsCorrection;
  final String? meaning;
  final String? comment;
}

double _readDouble(Object? value) {
  if (value is double) {
    return value;
  }
  if (value is num) {
    return value.toDouble();
  }
  return double.tryParse(value?.toString() ?? '') ?? 0;
}

int _readInt(Object? value) {
  if (value is int) {
    return value;
  }
  if (value is num) {
    return value.toInt();
  }
  return int.tryParse(value?.toString() ?? '') ?? 0;
}

String buildWordSampleText(WordPlaybackLanguage language) {
  return language.sampleWords.join('\n');
}

List<String> parseWordEntries(String rawText) {
  return rawText
      .split(RegExp(r'[\n,，;；]'))
      .map((item) => item.trim())
      .where((item) => item.isNotEmpty)
      .toList();
}
