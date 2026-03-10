class WeeklyStats {
  const WeeklyStats({
    required this.message,
    required this.days,
    this.insight,
  });

  factory WeeklyStats.fromJson(Map<String, dynamic> json) {
    return WeeklyStats(
      message: json['message']?.toString() ?? '',
      days: ((json['raw_stats'] as List<dynamic>? ?? const <dynamic>[]))
          .map((item) => WeeklyStatsDay.fromJson(item as Map<String, dynamic>))
          .toList(),
      insight: json['insights'] is Map<String, dynamic>
          ? WeeklyInsight.fromJson(json['insights'] as Map<String, dynamic>)
          : json['insights'] is Map
              ? WeeklyInsight.fromJson(
                  (json['insights'] as Map).map(
                    (key, value) => MapEntry(key.toString(), value),
                  ),
                )
              : null,
    );
  }

  final String message;
  final List<WeeklyStatsDay> days;
  final WeeklyInsight? insight;

  bool get hasData => days.isNotEmpty;

  int get totalTasks {
    return insight?.rawMetricTotal ??
        days.fold<int>(0, (sum, day) => sum + day.totalTasks);
  }

  int get completedTasks {
    return insight?.rawMetricCompleted ??
        days.fold<int>(0, (sum, day) => sum + day.completedTasks);
  }

  int get pendingTasks {
    return totalTasks - completedTasks;
  }

  int get completionRatePercent {
    if (totalTasks == 0) {
      return 0;
    }
    return (completedTasks * 100 / totalTasks).round();
  }
}

class WeeklyStatsDay {
  const WeeklyStatsDay({
    required this.date,
    required this.totalTasks,
    required this.completedTasks,
  });

  factory WeeklyStatsDay.fromJson(Map<String, dynamic> json) {
    if (json.containsKey('total_tasks') && json.containsKey('completed_tasks')) {
      return WeeklyStatsDay(
        date: json['date']?.toString() ?? '',
        totalTasks: _readInt(json['total_tasks']),
        completedTasks: _readInt(json['completed_tasks']),
      );
    }

    final tasks = (json['tasks'] as List<dynamic>? ?? const <dynamic>[]);
    final completedTasks = tasks
        .where(
          (item) =>
              item is Map<String, dynamic>
                  ? item['completed'] == true
                  : item is Map
                      ? item['completed'] == true
                      : false,
        )
        .length;

    return WeeklyStatsDay(
      date: json['date']?.toString() ?? '',
      totalTasks: tasks.length,
      completedTasks: completedTasks,
    );
  }

  final String date;
  final int totalTasks;
  final int completedTasks;

  int get pendingTasks => totalTasks - completedTasks;
}

class WeeklyInsight {
  const WeeklyInsight({
    required this.summary,
    required this.strengths,
    required this.areasForImprovement,
    required this.psychologicalInsight,
    required this.rawMetricTotal,
    required this.rawMetricCompleted,
  });

  factory WeeklyInsight.fromJson(Map<String, dynamic> json) {
    return WeeklyInsight(
      summary: json['summary']?.toString() ?? '',
      strengths: (json['strengths'] as List<dynamic>? ?? const <dynamic>[])
          .map((item) => item?.toString() ?? '')
          .where((item) => item.trim().isNotEmpty)
          .toList(),
      areasForImprovement:
          (json['areas_for_improvement'] as List<dynamic>? ?? const <dynamic>[])
              .map((item) => item?.toString() ?? '')
              .where((item) => item.trim().isNotEmpty)
              .toList(),
      psychologicalInsight:
          json['psychological_insight']?.toString() ?? '',
      rawMetricTotal: _readInt(json['raw_metric_total']),
      rawMetricCompleted: _readInt(json['raw_metric_completed']),
    );
  }

  final String summary;
  final List<String> strengths;
  final List<String> areasForImprovement;
  final String psychologicalInsight;
  final int rawMetricTotal;
  final int rawMetricCompleted;
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
