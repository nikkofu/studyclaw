enum VoiceCommandSurface { taskBoard, dictation }

extension VoiceCommandSurfaceX on VoiceCommandSurface {
  String get apiValue {
    switch (this) {
      case VoiceCommandSurface.taskBoard:
        return 'task_board';
      case VoiceCommandSurface.dictation:
        return 'dictation';
    }
  }

  String get label {
    switch (this) {
      case VoiceCommandSurface.taskBoard:
        return '任务板';
      case VoiceCommandSurface.dictation:
        return '听写';
    }
  }

  List<String> get sampleUtterances {
    switch (this) {
      case VoiceCommandSurface.taskBoard:
        return const <String>[
          '数学订正好了',
          '一课一练做完了',
          '全部都好了',
        ];
      case VoiceCommandSurface.dictation:
        return const <String>[
          '好了',
          '下一个',
          '重播',
        ];
    }
  }
}

class VoiceCommandContext {
  const VoiceCommandContext({
    required this.surface,
    this.dictation,
    this.taskBoard,
    this.locale,
    this.examples = const <String>[],
    this.metadata = const <String, String>{},
  });

  final VoiceCommandSurface surface;
  final VoiceCommandDictationContext? dictation;
  final VoiceCommandTaskBoardContext? taskBoard;
  final String? locale;
  final List<String> examples;
  final Map<String, String> metadata;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'surface': surface.apiValue,
      if (dictation != null) 'dictation': dictation!.toJson(),
      if (taskBoard != null) 'task_board': taskBoard!.toJson(),
      if (locale != null && locale!.trim().isNotEmpty) 'locale': locale,
      if (examples.isNotEmpty) 'examples': examples,
      if (metadata.isNotEmpty) 'metadata': metadata,
    };
  }
}

class VoiceCommandDictationContext {
  const VoiceCommandDictationContext({
    this.sessionId,
    this.currentWord,
    this.currentIndex,
    this.totalItems,
    this.canNext = false,
    this.canPrevious = false,
    this.isCompleted = false,
    this.language,
    this.playbackMode,
  });

  final String? sessionId;
  final String? currentWord;
  final int? currentIndex;
  final int? totalItems;
  final bool canNext;
  final bool canPrevious;
  final bool isCompleted;
  final String? language;
  final String? playbackMode;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      if (sessionId != null && sessionId!.trim().isNotEmpty)
        'session_id': sessionId,
      if (currentWord != null && currentWord!.trim().isNotEmpty)
        'current_word': currentWord,
      if (currentIndex != null) 'current_index': currentIndex,
      if (totalItems != null) 'total_items': totalItems,
      'can_next': canNext,
      'can_previous': canPrevious,
      'is_completed': isCompleted,
      if (language != null && language!.trim().isNotEmpty) 'language': language,
      if (playbackMode != null && playbackMode!.trim().isNotEmpty)
        'playback_mode': playbackMode,
    };
  }
}

class VoiceCommandTaskBoardContext {
  const VoiceCommandTaskBoardContext({
    this.focusedSubject,
    required this.summary,
    this.subjects = const <VoiceCommandTaskSubject>[],
    this.groups = const <VoiceCommandTaskGroup>[],
    this.tasks = const <VoiceCommandTaskItem>[],
  });

  final String? focusedSubject;
  final VoiceCommandTaskBoardSummary summary;
  final List<VoiceCommandTaskSubject> subjects;
  final List<VoiceCommandTaskGroup> groups;
  final List<VoiceCommandTaskItem> tasks;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      if (focusedSubject != null && focusedSubject!.trim().isNotEmpty)
        'focused_subject': focusedSubject,
      'summary': summary.toJson(),
      if (subjects.isNotEmpty)
        'subjects': subjects.map((item) => item.toJson()).toList(),
      if (groups.isNotEmpty)
        'groups': groups.map((item) => item.toJson()).toList(),
      if (tasks.isNotEmpty)
        'tasks': tasks.map((item) => item.toJson()).toList(),
    };
  }
}

class VoiceCommandTaskBoardSummary {
  const VoiceCommandTaskBoardSummary({
    required this.total,
    required this.completed,
    required this.pending,
  });

  final int total;
  final int completed;
  final int pending;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'total': total,
      'completed': completed,
      'pending': pending,
    };
  }
}

class VoiceCommandTaskSubject {
  const VoiceCommandTaskSubject({
    required this.subject,
    required this.status,
    required this.completed,
    required this.pending,
    required this.total,
  });

  final String subject;
  final String status;
  final int completed;
  final int pending;
  final int total;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'subject': subject,
      'status': status,
      'completed': completed,
      'pending': pending,
      'total': total,
    };
  }
}

class VoiceCommandTaskGroup {
  const VoiceCommandTaskGroup({
    required this.subject,
    required this.groupTitle,
    required this.status,
    required this.completed,
    required this.pending,
    required this.total,
  });

  final String subject;
  final String groupTitle;
  final String status;
  final int completed;
  final int pending;
  final int total;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'subject': subject,
      'group_title': groupTitle,
      'status': status,
      'completed': completed,
      'pending': pending,
      'total': total,
    };
  }
}

class VoiceCommandTaskItem {
  const VoiceCommandTaskItem({
    required this.taskId,
    required this.subject,
    required this.groupTitle,
    required this.content,
    required this.completed,
    required this.status,
  });

  final int taskId;
  final String subject;
  final String groupTitle;
  final String content;
  final bool completed;
  final String status;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'task_id': taskId,
      'subject': subject,
      'group_title': groupTitle,
      'content': content,
      'completed': completed,
      'status': status,
    };
  }
}

class VoiceCommandResolution {
  const VoiceCommandResolution({
    required this.action,
    required this.reason,
    required this.parserMode,
    required this.confidence,
    required this.normalizedTranscript,
    required this.surface,
    required this.target,
  });

  factory VoiceCommandResolution.fromJson(Map<String, dynamic> json) {
    return VoiceCommandResolution(
      action: json['action']?.toString() ?? 'none',
      reason: json['reason']?.toString() ?? '',
      parserMode: json['parser_mode']?.toString() ?? 'rule_fallback',
      confidence: _readDouble(json['confidence']),
      normalizedTranscript: json['normalized_transcript']?.toString() ?? '',
      surface: _surfaceFromString(json['surface']?.toString()),
      target: VoiceCommandTarget.fromJson(
        json['target'] as Map<String, dynamic>? ?? const <String, dynamic>{},
      ),
    );
  }

  final String action;
  final String reason;
  final String parserMode;
  final double confidence;
  final String normalizedTranscript;
  final VoiceCommandSurface surface;
  final VoiceCommandTarget target;

  bool get hasExecutableAction => action != 'none';

  String get actionLabel {
    switch (action) {
      case 'dictation_next':
        return '切到下一词';
      case 'dictation_previous':
        return '返回上一词';
      case 'dictation_replay':
        return '重播当前词';
      case 'task_complete_item':
        return '完成单个任务';
      case 'task_complete_group':
        return '完成任务分组';
      case 'task_complete_subject':
        return '完成整科学任务';
      case 'task_complete_all':
        return '完成全部任务';
      default:
        return '暂不执行';
    }
  }
}

class VoiceCommandTarget {
  const VoiceCommandTarget({
    this.sessionId,
    this.taskId,
    this.subject,
    this.groupTitle,
    this.taskContent,
  });

  factory VoiceCommandTarget.fromJson(Map<String, dynamic> json) {
    return VoiceCommandTarget(
      sessionId: _readString(json['session_id']),
      taskId: _readIntOrNull(json['task_id']),
      subject: _readString(json['subject']),
      groupTitle: _readString(json['group_title']),
      taskContent: _readString(json['task_content']),
    );
  }

  final String? sessionId;
  final int? taskId;
  final String? subject;
  final String? groupTitle;
  final String? taskContent;
}

VoiceCommandSurface _surfaceFromString(String? value) {
  switch (value) {
    case 'dictation':
      return VoiceCommandSurface.dictation;
    case 'task_board':
    default:
      return VoiceCommandSurface.taskBoard;
  }
}

double _readDouble(Object? value) {
  if (value is double) return value;
  if (value is int) return value.toDouble();
  if (value is num) return value.toDouble();
  return double.tryParse(value?.toString() ?? '') ?? 0;
}

int? _readIntOrNull(Object? value) {
  if (value == null) return null;
  if (value is int) return value;
  if (value is num) return value.toInt();
  return int.tryParse(value.toString());
}

String? _readString(Object? value) {
  final result = value?.toString().trim() ?? '';
  return result.isEmpty ? null : result;
}
