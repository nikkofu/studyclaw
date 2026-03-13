class TaskBoardRequest {
  const TaskBoardRequest({
    required this.apiBaseUrl,
    required this.familyId,
    required this.userId,
    required this.date,
  });

  final String apiBaseUrl;
  final int familyId;
  final int userId;
  final String date;

  TaskBoardRequest copyWith({
    String? apiBaseUrl,
    int? familyId,
    int? userId,
    String? date,
  }) {
    return TaskBoardRequest(
      apiBaseUrl: apiBaseUrl ?? this.apiBaseUrl,
      familyId: familyId ?? this.familyId,
      userId: userId ?? this.userId,
      date: date ?? this.date,
    );
  }

  String? validate() {
    final trimmedApiBaseUrl = apiBaseUrl.trim();
    final parsedApiUri = Uri.tryParse(trimmedApiBaseUrl);

    if (trimmedApiBaseUrl.isEmpty) {
      return 'API 地址不能为空。';
    }
    if (parsedApiUri == null ||
        parsedApiUri.scheme.isEmpty ||
        parsedApiUri.host.isEmpty) {
      return 'API 地址需要是有效的 http/https URL。';
    }
    if (parsedApiUri.scheme != 'http' && parsedApiUri.scheme != 'https') {
      return 'API 地址需要以 http:// 或 https:// 开头。';
    }
    if (familyId <= 0) {
      return 'family_id 需要是正整数。';
    }
    if (userId <= 0) {
      return 'user_id 需要是正整数。';
    }
    if (!RegExp(r'^\d{4}-\d{2}-\d{2}$').hasMatch(date)) {
      return '日期需要是 YYYY-MM-DD。';
    }
    if (parseTaskBoardDate(date) == null) {
      return '日期不是有效的自然日。';
    }
    return null;
  }
}

class TaskBoard {
  const TaskBoard({
    required this.date,
    required this.tasks,
    required this.groups,
    required this.homeworkGroups,
    required this.summary,
    this.message,
  });

  factory TaskBoard.fromJson(Map<String, dynamic> json) {
    return TaskBoard(
      date: json['date']?.toString() ?? '',
      message: json['message']?.toString(),
      tasks: ((json['tasks'] as List<dynamic>? ?? const <dynamic>[]))
          .map((item) => TaskItem.fromJson(item as Map<String, dynamic>))
          .toList(),
      groups: ((json['groups'] as List<dynamic>? ?? const <dynamic>[]))
          .map((item) => TaskGroup.fromJson(item as Map<String, dynamic>))
          .toList(),
      homeworkGroups: ((json['homework_groups'] as List<dynamic>? ??
              const <dynamic>[]))
          .map((item) => HomeworkGroup.fromJson(item as Map<String, dynamic>))
          .toList(),
      summary: BoardSummary.fromJson(
        json['summary'] as Map<String, dynamic>? ?? const {},
      ),
    );
  }

  final String date;
  final String? message;
  final List<TaskItem> tasks;
  final List<TaskGroup> groups;
  final List<HomeworkGroup> homeworkGroups;
  final BoardSummary summary;
}

class TaskItem {
  const TaskItem({
    required this.taskId,
    required this.subject,
    required this.groupTitle,
    required this.content,
    required this.completed,
    required this.status,
    this.taskType = '',
    this.referenceTitle = '',
    this.referenceAuthor = '',
    this.referenceText = '',
    this.hideReferenceFromChild = false,
    this.analysisMode = '',
  });

  factory TaskItem.fromJson(Map<String, dynamic> json) {
    return TaskItem(
      taskId: _readInt(json['task_id']),
      subject: json['subject']?.toString() ?? '未分类',
      groupTitle: _readString(json['group_title']).isNotEmpty
          ? _readString(json['group_title'])
          : _readString(json['content']),
      content: json['content']?.toString() ?? '',
      completed: json['completed'] == true,
      status: json['status']?.toString() ?? 'pending',
      taskType: _readString(json['task_type']),
      referenceTitle: _readString(json['reference_title']),
      referenceAuthor: _readString(json['reference_author']),
      referenceText: _readString(json['reference_text']),
      hideReferenceFromChild: json['hide_reference_from_child'] == true,
      analysisMode: _readString(json['analysis_mode']),
    );
  }

  final int taskId;
  final String subject;
  final String groupTitle;
  final String content;
  final bool completed;
  final String status;
  final String taskType;
  final String referenceTitle;
  final String referenceAuthor;
  final String referenceText;
  final bool hideReferenceFromChild;
  final String analysisMode;

  bool get hasReferenceMaterial => referenceText.trim().isNotEmpty;
}

class HomeworkGroup {
  const HomeworkGroup({
    required this.subject,
    required this.groupTitle,
    required this.total,
    required this.completed,
    required this.pending,
    required this.status,
  });

  factory HomeworkGroup.fromJson(Map<String, dynamic> json) {
    return HomeworkGroup(
      subject: json['subject']?.toString() ?? '未分类',
      groupTitle: json['group_title']?.toString() ?? '',
      total: _readInt(json['total']),
      completed: _readInt(json['completed']),
      pending: _readInt(json['pending']),
      status: json['status']?.toString() ?? 'pending',
    );
  }

  final String subject;
  final String groupTitle;
  final int total;
  final int completed;
  final int pending;
  final String status;
}

class TaskGroup {
  const TaskGroup({
    required this.subject,
    required this.total,
    required this.completed,
    required this.pending,
    required this.status,
  });

  factory TaskGroup.fromJson(Map<String, dynamic> json) {
    return TaskGroup(
      subject: json['subject']?.toString() ?? '未分类',
      total: _readInt(json['total']),
      completed: _readInt(json['completed']),
      pending: _readInt(json['pending']),
      status: json['status']?.toString() ?? 'pending',
    );
  }

  final String subject;
  final int total;
  final int completed;
  final int pending;
  final String status;
}

class BoardSummary {
  const BoardSummary({
    required this.total,
    required this.completed,
    required this.pending,
    required this.status,
  });

  factory BoardSummary.fromJson(Map<String, dynamic> json) {
    return BoardSummary(
      total: _readInt(json['total']),
      completed: _readInt(json['completed']),
      pending: _readInt(json['pending']),
      status: json['status']?.toString() ?? 'empty',
    );
  }

  final int total;
  final int completed;
  final int pending;
  final String status;

  String get statusLabel {
    switch (status) {
      case 'completed':
        return '全部完成';
      case 'partial':
        return '部分完成';
      case 'pending':
        return '待开始';
      default:
        return '空任务板';
    }
  }
}

DateTime? parseTaskBoardDate(String value) {
  try {
    final parsedDate = DateTime.parse(value);
    return DateTime(parsedDate.year, parsedDate.month, parsedDate.day);
  } on FormatException {
    return null;
  }
}

String formatTaskBoardDate(DateTime value) {
  final year = value.year.toString().padLeft(4, '0');
  final month = value.month.toString().padLeft(2, '0');
  final day = value.day.toString().padLeft(2, '0');
  return '$year-$month-$day';
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

String _readString(Object? value) {
  return value?.toString() ?? '';
}
